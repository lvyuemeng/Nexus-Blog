+++
date = '2025-05-30T11:54:44+08:00'
draft = false
title = 'Ariel Embassy'
tag = ["embed", "ariel", "embassy"]
author = ["nostalgia"]
+++

# Embassy on Ariel-OS

## Executor Types

They are multiple ways to implement and integrate embassy functionality.

Embassy is a runtime existing in whole life time of program. Thus any async execution will be entrapped in runtime scheduling.

---

First is the most simple, we directly entrap everything in such runtime scheduling. This means a sole thread.

```rust
#[cfg(feature = "executor-single-thread")]
#[unsafe(export_name = "__ariel_os_embassy_init")]
fn init() -> ! {
    use static_cell::StaticCell;

    debug!("ariel-os-embassy::init(): using single thread executor");
    let p = hal::init();

    static EXECUTOR: StaticCell<hal::Executor> = StaticCell::new();
    EXECUTOR
        .init_with(|| hal::Executor::new())
        .run(|spawner| spawner.must_spawn(init_task(p)))
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L145-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/lib.rs#L145-L155)

Above code exported as a entry point. It create a static life time instance and then run it.

A normal executor have the lowest priority thus any interrupt will preempt tasks running on the normal executor.

---

If we want to integrate threads system, we should take in account the relation between it. We could regard it equally. That means independent interruption reason to dispatch two system.

we serve a unused hardware or prescribed `SWI` software interruption for executor called `InterruptExecutor`. Which allows user to define relative priorities.

Thus, after setup, it contains a `on_interrupt` that should be called by potential designed `SWI`.

Ariel achieve this by declaration macro to register specified one in embassy:
```rust
/// ```Rust
/// executor_swi!(SWI_IRQ_1);
/// ```
///
/// into this:
///
/// ```Rust
/// pub use interrupt::SWI_IRQ_1 as SWI;
/// #[interrupt]
/// unsafe fn SWI_IRQ_1() {
///     unsafe { crate::EXECUTOR.on_interrupt() }
/// }
/// ```
///
/// Note: this expects the `interrupt` to be present (e.g., "used") and that it contains the ISR
/// type.
#[macro_export]
macro_rules! executor_swi {
    ($swi:ident) => {
        pub use interrupt::$swi as SWI;
        #[interrupt]
        unsafe fn $swi() {
            // SAFETY:
            // - As required, it is called from an ISR
            // - The interrupt is enabled by start(), thus this is not called before start.
            //   (This macro just adds "only enable it after starting the executor" to the
            //   requirements of the unsafe interrupt starting; the safe start() function
            //    trusts the user to pass the right number.)
            unsafe { crate::EXECUTOR.on_interrupt() }
        }
    };
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L27-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/58a29321be20b50bf018d6e7917cc7aeac4d1c14/src/ariel-os-embassy-common/src/executor_swi.rs#L27-L42)

Thus:

```rust
#[cfg(feature = "executor-interrupt")]
#[distributed_slice(ariel_os_rt::INIT_FUNCS)]
pub(crate) fn init() {
    debug!("ariel-os-embassy::init(): using interrupt mode executor");
    let p = hal::init();

    #[cfg(any(context = "nrf", context = "rp", context = "stm32"))]
    {
        hal::EXECUTOR.start(hal::SWI);
        hal::EXECUTOR.spawner().must_spawn(init_task(p));
    }

    #[cfg(context = "esp")]
    EXECUTOR.run(|spawner| spawner.must_spawn(init_task(p)));
}
```

Here after spawn the task we doesn't go to `run` rather we first start the `SWI` and then schedule independently. 

Ariel-OS use procedural macro by `linkme` to register the function on `INIT_FUNCS` rather export it on boot process.

The usual usage of `linkme` is to declare a array:
```rust
use linkme::distributed_slice;

#[distributed_slice]
pub static INIT_FUNCS: [fn()] = [..];
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L83-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/58a29321be20b50bf018d6e7917cc7aeac4d1c14/src/ariel-os-rt/src/lib.rs#L83-L86)

After that, we can register function by above macro with identifier such as `$crate::ident`.

---

The third way is to consider the strategy to insert whole embassy runtime into the thread system. We can construct a independent runtime thread to incorporate preemption behavoir into embassy runtime.

Ariel-OS achieve it with a flag design. In scheduler of threads, every thread has a flag indication that could be used for parking by `set_flag` and `wait`.

```rust
/// Sets flags for a thread.
///
/// If the thread was blocked on these flags it's unblocked and added
/// to the runqueue.
///
/// # Panics
///
/// Panics if `thread_id` is >= [`THREAD_COUNT`](crate::THREAD_COUNT).
pub fn set(thread_id: ThreadId, mask: ThreadFlags) {
    SCHEDULER.with_mut(|mut scheduler| scheduler.flag_set(thread_id, mask));
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L17-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-threads/src/thread_flags.rs#L17-L26)

```rust
/// Waits until any flag in `mask` is set for the current thread.
///
/// Returns all set flags for this mask and clears them for the thread.
///
/// # Panics
///
/// Panics if this is called outside of a thread context.
pub fn wait_any(mask: ThreadFlags) -> ThreadFlags {
    loop {
        if let Some(flags) = SCHEDULER.with_mut(|mut scheduler| scheduler.flag_wait_any(mask)) {
            return flags;
        }
    }
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L44-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-threads/src/thread_flags.rs#L44-L57)

We know executor use `__pender(...)` to wake itself to poll. Indeed, we can therefore `set` in `__pender`. And in `run`, we `wait_any` after `poll` to park itself.

```rust
// This is only used between `__pender` and `Executor::run( )`, actual flag
// doesn't matter.
const THREAD_FLAG_WAKEUP: ThreadFlags = 0x01;

// SAFETY: this name is required by embassy-executor and the function signature matches the
// expected one.
#[unsafe(no_mangle)]
fn __pender(context: *mut ()) {
    // SAFETY: `context` is a `ThreadId` passed by `ThreadExecutor::new`.
    let thread_id = ThreadId::new(context as usize as u8);

    thread_flags::set(thread_id, THREAD_FLAG_WAKEUP);
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L18-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/thread_executor.rs#L18-L23)

```rust
    pub fn run(&'static mut self, init: impl FnOnce(Spawner)) -> ! {
        init(self.inner.spawner());

        loop {
            // SAFETY: `poll()` may net be called reentrantly on the same executor, which we don't.
            unsafe {
                self.inner.poll();
            };
            thread_flags::wait_any(THREAD_FLAG_WAKEUP);
        }
    }
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L68-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/thread_executor.rs#L68-L79)

Ariel-OS use procedural macro to initiate a thread.

```rust
#[cfg(feature = "executor-thread")]
#[ariel_os_macros::thread(autostart, no_wait, stacksize = executor_thread::STACKSIZE, priority = executor_thread::PRIORITY)]
fn init() {
    use static_cell::StaticCell;

    debug!(
        "ariel-os-embassy::init(): using thread executor with thread stack size {}",
        executor_thread::STACKSIZE
    );
    let p = hal::init();

    static EXECUTOR: StaticCell<thread_executor::Executor> = StaticCell::new();
    EXECUTOR
        .init_with(thread_executor::Executor::new)
        .run(|spawner| spawner.must_spawn(init_task(p)));
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L159-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/lib.rs#L159-L172)

The only difference is we initiate in a thread. The key of the macro is:

```rust
#[proc_macro_attribute]
pub fn thread(args: TokenStream, item: TokenStream) -> TokenStream {
    ...
    let expanded = quote! {
        #[inline(always)]
        #thread_function

        #[allow(non_snake_case)]
        fn #trampoline_function_name() {
            #maybe_wait_for_start_event;
            #fn_name()
        }

        #thread_crate::autostart_thread!(#trampoline_function_name, stacksize = #stack_size, priority = #priority, affinity = #affinity);
    };

    TokenStream::from(expanded)
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L81-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/58a29321be20b50bf018d6e7917cc7aeac4d1c14/src/ariel-os-macros/src/thread.rs#L81-L94)

`autostart_thread!` actually use scheduler to initiate the thread.

---

## Task Initiation

Above we only spawn `init_task` to handle all things. We put these underneath suggesting user can't initiate its own executor. Rather it can only use spawner to spawn if possible.

In embassy, `spawner` is actually a `executor` wrapper to handle `SpawnToken` generated by `embassy_executor::task`.

We can extract the `spawner` as a global static instance.

However, it means if user spawn a task in a thread, it should be sended to the thread where executor located or the interrupt stack space. Which should implement `Send` unless the task can't be sended.

We has such instance in embassy:
```rust
/// Handle to spawn tasks into an executor from any thread.
///
/// This Spawner can be used from any thread (it is Send), but it can
/// only spawn Send tasks. The reason for this is spawning is effectively
/// "sending" the tasks to the executor thread.
///
/// If you want to spawn non-Send tasks, use [Spawner].
#[derive(Copy, Clone)]
pub struct SendSpawner {
    executor: &'static raw::SyncExecutor,
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L239-blue?style=flat-square&logo=github)](https://github.com/embassy-rs/embassy/blob/b024d5e892618947f81efd72f2c4224ae830d891/embassy-executor/src/spawner.rs#L239-L242)

```rust
pub(crate) static SPAWNER: CriticalSectionMutex<OnceCell<SendSpawner>> =
    CriticalSectionMutex::new(OnceCell::new());

/// Gets a spawner for the system executor.
///
/// # Panics
///
/// Panics when called before the system has finished initializing.
pub fn spawner() -> SendSpawner {
    SPAWNER.lock(|x| *x.get().unwrap())
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L21-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/asynch.rs#L21-L31)

Allocation of new task instance is workable if we extract the spawner of our executor into global spawner.

```rust
#[embassy_executor::task]
#[allow(clippy::too_many_lines)]
async fn init_task(mut peripherals: hal::OptionalPeripherals) {
    let spawner = asynch::Spawner::for_current_executor().await;
    asynch::set_spawner(spawner.make_send());
    ...
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L176-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/lib.rs#L176-#178)

`autostart` is a parameter to tell system automatically start specified function. Usually used for peripherals manipulation.

In thread, we initiate in scheduler so `autostart` is mandatory as `autostart_thread!` suggests.

Then, we can follow the routine as `INIT_FUNC` to create `EMBASSY_TASKS`.

```rust
/// # Parameters
///
/// - `autostart`: (*optional*) run the task at startup; required to use `peripherals` and/or
///   hooks.
///     - `peripherals`: (*optional*) provide the function with a peripheral struct as the first
///       parameter.
///       The `peripherals` parameter can only be used on `autostart` tasks.
///       The peripheral struct must be defined with the `ariel_os::hal::define_peripherals!`
///       macro.
///     - hooks: (*optional*) available hooks are:
///         - `usb_builder_hook`: when present, the macro will define a static `USB_BUILDER_HOOK`
///           of type `UsbBuilderHook`, allowing to access and modify the system-provided
///           `embassy_usb::Builder` through `Delegate::with()`, *before* it is built by the system.
/// - `pool_size`: (*optional*) set the maximum number of concurrent tasks that can be spawned for
///   the function (defaults to `1`).
///   Cannot be used on `autostart` tasks.
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L7-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/58a29321be20b50bf018d6e7917cc7aeac4d1c14/src/ariel-os-macros/src/task.rs#L7-L22)

Thus `task` macro dispatch a special routine for `autostart` rather than directly initiate a task by embassy macro, it will again register function into `EMBASSY_TASKS`.

```rust
#[proc_macro_attribute]
pub fn task(args: TokenStream, item: TokenStream) -> TokenStream {
    ...
        quote! {
            #delegates

            #[allow(non_snake_case)]
            #[#ariel_os_crate::reexports::linkme::distributed_slice(#ariel_os_crate::EMBASSY_TASKS)]
            #[linkme(crate = #ariel_os_crate::reexports::linkme)]
            fn #new_function_name(
                spawner: #ariel_os_crate::asynch::Spawner,
                mut peripherals: &mut #ariel_os_crate::hal::OptionalPeripherals,
            ) {
                use #ariel_os_crate::hal::TakePeripherals;
                let task = #task_function_name(#peripheral_param);
                spawner.spawn(task).unwrap();
            }

            #[#ariel_os_crate::reexports::embassy_executor::task(embassy_executor = #ariel_os_crate::reexports::embassy_executor)]
            #task_function
        }
    ...
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L95-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/58a29321be20b50bf018d6e7917cc7aeac4d1c14/src/ariel-os-macros/src/task.rs#L95-L111)

Which we use a callback function.

In `init_task`, we call each in the `EMBASSY_TASKS` array.

```rust
#[embassy_executor::task]
#[allow(clippy::too_many_lines)]
async fn init_task(mut peripherals: hal::OptionalPeripherals) {
    ...
    // Tasks have to be started before driver initializations so that the tasks are able to
    // configure the drivers using hooks.
    for task in EMBASSY_TASKS {
        task(spawner, &mut peripherals);
    }
    ...
}
```

[![View Source Code](https://img.shields.io/badge/Source_Code-L210-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/lib.rs#L210-L212)

---

## Peripherals

Peripherals are exclusive outer resources abstraction.

Ariel-OS use `ZSTs`(zero size types) to represent these.

We can define peripherals by given hardware concrete resources.

```rust
#[cfg(context = "nrf52840")]
pub type SensorSpi = spi::main::SPI3;
#[cfg(context = "nrf52840")]
ariel_os::hal::define_peripherals!(Peripherals {
    spi_sck: P1_15,
    spi_miso: P1_14,
    spi_mosi: P1_13,
    spi_cs: P1_12,
});
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L22-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/80619c2e/tests/spi-main/src/pins.rs#L22-L30)

Each is a special field can only be taken once:

```rust
macro_rules! define_peripherals {
    ...
        impl $crate::TakePeripherals<$peripherals> for &mut $crate::OptionalPeripherals {
            fn take_peripherals(&mut self) -> $peripherals {
                $peripherals {
                    $(
                        $(#[$inner])*
                        $peripheral_name: self.$peripheral_field.take().unwrap()
                    ),*
                }
            }
        }
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L42-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/80619c2e/src/ariel-os-hal/src/define_peripherals.rs#L42-L51)

So what does `hal::init()` do in previous code? Actually it routes to `hal` crate which only compiles specified concrete hardware crate containing the initialization of `OptionalPeripherals` solely defined for its hardware resource.

```rust
#[doc(hidden)]
/// Dummy type.
pub struct Peripherals;

impl From<Peripherals> for OptionalPeripherals {
    fn from(_peripherals: Peripherals) -> Self {
        Self {}
    }
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L61-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-hal/src/dummy/mod.rs#L61-L70)

Now we can see the full view of `init_task()`.

We implement HAL initialization process for predefined peripherals, s.t.:

```rust
#[embassy_executor::task]
#[allow(clippy::too_many_lines)]
async fn init_task(mut peripherals: hal::OptionalPeripherals) {
    // Move out the peripherals required for drivers, so that tasks cannot mistakenly take them.
    #[cfg(feature = "usb")]
    let usb_peripherals = hal::usb::Peripherals::new(&mut peripherals);
    ...
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L205-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/lib.rs#L205-L206)

Some of them is a task should be spawned such as `net`.

```rust
#[embassy_executor::task]
#[allow(clippy::too_many_lines)]
async fn init_task(mut peripherals: hal::OptionalPeripherals) {
        let config = net::config();

        let seed = net::unique_seed();
        debug!("Network stack seed: {:#x}", seed);

        // Init network stack
        let (stack, runner) = embassy_net::new(
            device,
            config,
            RESOURCES.init_with(StackResources::new),
            seed,
        );

        spawner.spawn(net::net_task(runner)).unwrap();
    ...
}
```
[![View Source Code](https://img.shields.io/badge/Source_Code-L328-blue?style=flat-square&logo=github)](https://github.com/ariel-os/ariel-os/blob/main/src/ariel-os-embassy/src/lib.rs#L328-L341)


