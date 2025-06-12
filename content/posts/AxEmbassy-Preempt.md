+++
date = '2025-06-12T11:23:15+08:00'
draft = false
title = 'AxEmbassy Preempt'
tag = ["embed", "arceos", "embassy"]
author = ["nostalgia"]
+++

# AxEmbassy-Preempt

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

## Usuage

In such development we can't use macro as usual because each task should be wrapped with a `PrioFuture`, as `PrioFuture<impl Future<Output = ...>>`, and task pool can only infer for original task type.

We manually show this:
```rust
fn async_tick_raw(
    id: u64,
    millis: u64,
    busy_iters: u64,
    prio: u8,
) -> embassy_executor::SpawnToken<impl Sized> {
    type Fut = PrioFuture<impl ::core::future::Future + 'static>;
    const POOL_SIZE:usize = 5;
    static POOL: embassy_executor::raw::TaskPool<Fut,POOL_SIZE> = embassy_executor::raw::TaskPool::new();

    let prio_fut = PrioFuture::new(async_tick(id, millis, busy_iters), prio);
    unsafe { POOL.spawn(|| prio_fut) }
}
```
If we want user seamlessly assign priority, it should be a pre-work on procedural macro.