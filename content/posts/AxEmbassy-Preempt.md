+++
date = '2025-06-12T11:23:15+08:00'
draft = false
title = 'AxEmbassy Preempt'
tag = ["embed", "arceos", "embassy"]
author = ["nostalgia"]
+++

## Development

Embassy use head insertion to add a new task, adn poll every task in sequence. In order to add preemption, we can't modify embassy itself due to self-referent structure. Another strategy is to modify `Future` itself with a priority system to change the probability of polling distribution.


To ensure a trace of future, we use `TaskId`. After a initiation of future, we insert it in `Scheduler`.

```rust
struct PrioScheduler {
	// u64 is the count of task in given prio level.
    prios: BTreeMap<Prio, u64>,
    tasks: BTreeMap<TaskId, Prio>,
}
```

```rust
impl<F: Future> Future for PrioFuture<F> {
	...
    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        let this = unsafe { self.get_unchecked_mut() };

        // Register on first poll
        if !this.registered {
            SCHEDULER.lock().insert(this.prio, this.id);
            this.registered = true;
        }
		...
    }
}
```

A given task has a priority level, we use reverse order, higher priority corresponds to smaller number.

```rust
pub type Prio = u8;
pub const MAX_PRIO: Prio = 31;
```

So in every loop, each task will be polled equally, we want high priority task be polled first is impossible. Therefore, we can only achieve a high priority check on each poll.

```rust
impl<F: Future> Future for PrioFuture<F> {
	...
    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
		...
        let highest_prio = SCHEDULER.lock().cur_prio().unwrap_or(MAX_PRIO);

        if this.prio > highest_prio {
            cx.waker().wake_by_ref()
            return Poll::Pending;
        }
		...
	}
}
```

With a peek on priority:

```rust
impl PrioScheduler {
    pub fn cur_prio(&self) -> Option<Prio> {
        self.prios
            .iter()
            .filter(|(_, cnt)| **cnt > 0)
            .next()
            .map(|(prio, _)| *prio)
    }
}
```

We see that if the highest priority isn't dynamic, it will always point to the most highest priority task only, and block everything. So we should introduce a dynamic system to achieve this.

If a task pass the priority check, it must be a high priority to average level, if we poll the task resulting `Pending`, we can ensure that in a short amount of time it's `Pending` too, so we remove it from running priority queue to ease the current priority check level. Then if it's `Ready`, we can ensure that it should run itself now, we push back its priority to running priority queue, increase the priority check level.

```rust
impl<F: Future> Future for PrioFuture<F> {
	...
    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
		...
        let inner = unsafe { Pin::new_unchecked(&mut this.inner) };
        match inner.poll(cx) {
            Poll::Ready(output) => {
                let id = this.id;
                SCHEDULER.lock().unpark_task_prio(id);
                Poll::Ready(output)
            }
            Poll::Pending => {
                let id = this.id;
                SCHEDULER.lock().park_task_prio(id);
                Poll::Pending
            }
        }
    }
}


impl PrioScheduler {
    pub fn park_task_prio(&mut self, id: TaskId) {
        if let Some(prio) = self.tasks.get(&id) {
            self.prios.entry(*prio).and_modify(|cnt| {
                *cnt = cnt.saturating_sub(1);
            });
        }
    }

    pub fn unpark_task_prio(&mut self, id: TaskId) {
        if let Some(prio) = self.tasks.get(&id) {
            self.prios.entry(*prio).and_modify(|cnt| {
                *cnt = cnt.saturating_add(1);
            });
        }
    }
}
```

So if a task with high priority and more frequent `Ready` a.k.a tense work, will always keep the priority on higher level, allowing a specific selection on high priority task. Rather, if it's more frequent in `Pending`, a.k.a loose work, will ease the priority check level, allowing other lower priority task to run.

## Improvement

In above design, I found the including factor isn't complete enough. Especially the priority boundary check only depends on the counts of highest task. So I try to extend to 2 decimal higher and including priority and task counts.

We use a scaling factor to extend:

```rust
/// Represents a user-defined priority, ranging from 0 to 255.
type UserPrio = u8;
const MIN_USER_PRIO: UserPrio = u8::MAX;
const MAX_USER_PRIO: UserPrio = u8::MIN;

/// A fixed-point representation of priority to handle fractional values.
/// This type ensures consistent scaling and avoids magic numbers in calculations.
#[derive(Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Debug)]
struct Prio(u64);

impl From<UserPrio> for Prio {
    fn from(value: UserPrio) -> Self {
        // Scale the user-priority to the internal fixed-point representation.
        Self((value as u64).saturating_mul(Prio::SCALING_FACTOR))
    }
}
```

Thus, every check has a tolerance proportional to the task priority:

```rust
impl<F: Future> Future for PrioFuture<F> {
    ...
    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        ...
        let cur_prio = s.cur_prio;
        let prio = this.prio;
        let tol = Prio::weight(prio, Prio::TOL);

        s.adjust_cur_prio(prio);

        // If future prio > cur_prio + tolerance, park it
        let threshold: u64 = prio.as_u64().saturating_sub(cur_prio);
        if threshold > tol {
            // info!("prio: {}, cur_prio: {}", this.prio, s.cur_prio);
            s.park_future(this.id);
            cx.waker().wake_by_ref();
            return Poll::Pending;
        } else {
            s.unpark_future(this.id);
        }
    }
}

/// Applies a weight to a normalized factor.
pub fn weight<F: Into<u64>, G: Into<u64>>(factor: F, weight: G) -> u64 {
    factor
        .into()
        .saturating_mul(weight.into())
        .div_euclid(Prio::CLAMP_MAX)
}
```

Where we use `weight` to achieve the percent proportion.

To adjust the priority adjustment, we suppose each task has affinity to pull the current priority level to themselves, for lower priority(a.k.a huger number), and smaller counts of active tasks in its priority level, must has greater affinity to avoid starving. Thus, we use a parameterization by `cur_prio = cur_prio + (task_prio - cur_prio) * affinity`, `0 < affinity < 1` -> `0 < affinity < 100` in 2 decimal extension.

We use normalization, that's, choose the higher bound and lower bound to acquire percent factor. Then for factor of active task, and priority, we acquire such factor and sum them in a balance weight. Finally we add it to the `cur_prio`.

```rust
impl PrioScheduler {
    pub fn adjust_cur_prio(&mut self, prio: Prio) {
        let (active, all) = self.get_prio_status(prio);
        let (_, norm_active) = Prio::norm_factor((all as u64, 0), active as u64);

        let (highest, lowest) = self.prio_range();
        let (norm_prio, _) = Prio::norm_factor((highest, lowest), prio);

        let prio: u64 = prio.into();
        let cur_prio = self.cur_prio.into();
        let factor = (Prio::weight(norm_prio, Prio::PRIO_EFFECT)
            .saturating_add(Prio::weight(norm_active, Prio::ACTIVE_EFFECT)))
        .clamp(Prio::CLAMP_MIN, Prio::CLAMP_MAX);
        if prio > cur_prio {
            self.cur_prio += Prio::weight(prio - cur_prio, factor);
        } else {
            self.cur_prio -= Prio::weight(cur_prio - prio, factor);
        }
    }
}

/// Normalizes a current value within a given bound to a range [CLAMP_MIN_FACTOR, CLAMP_MAX_FACTOR].
///
/// # Returns
/// `(norm_pos, norm_neg)` where:
/// - `norm_pos` indicates closeness to the lower bound (higher for values closer to low).
/// - `norm_neg` indicates closeness to the upper bound (higher for values closer to high).
pub fn norm_factor<F: Into<u64>, G: Into<u64>>(bound: (F, F), cur: G) -> (u64, u64) {
    let (mut low, mut high) = (bound.0.into(), bound.1.into());
    if low > high {
        core::mem::swap(&mut low, &mut high);
    }
    let cur = cur.into();

    if high == low {
        // Avoid division by zero if bounds are the same
        return (Prio::CLAMP_MAX, Prio::CLAMP_MAX);
    }

    let range = high - low;
    let norm_pos = ((cur - low).saturating_mul(Prio::CLAMP_MAX) / range)
        .clamp(Prio::CLAMP_MIN, Prio::CLAMP_MAX);
    let norm_neg = ((high - cur).saturating_mul(Prio::CLAMP_MAX) / range)
        .clamp(Prio::CLAMP_MIN, Prio::CLAMP_MAX);
    (norm_pos, norm_neg)
}
```

After adjust factor, we _park_ the future, to reduce active count, even it must be queued again necessarily. But it's not in our priority boundary check region.

## Usuage

It's rather simple to just warp your want code into the `PrioFuture`. It's dynamical for whole task or a small code block.

```rust
if busy_iters > 0 {
    let _res = black_box(PrioFuture::new(async_busy_work(busy_iters), $prios).await);
}
```