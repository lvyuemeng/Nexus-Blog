+++
date = '2025-09-20T19:12:58+08:00'
draft = false
title = 'Ipc Spec'
tag = ["ipc"]
author = ["nostalgia"]
+++

## Philosophy

Separates the object instances (which are process-specific) from the data they manage (which is shared). This design allows each process to have its own interface objects while manipulating the same underlying shared memory structures through atomic operations.

Each process initiates the different object instances with the same memory data. We should use the spin, atomic and any inter-process synchronization to ensure this.

- No Ownership: The handle doesn't *own* the shared data
- Pointer-Based: Use `NonNull<T>` to reference shared memory
- Safe Drop: Only cleanup process-local resources, never touch shared memory
- Multiple Handles: Many processes can have handles to the same queue data

## Memory Layout

Refer `Claude`
```bash
SHARED MEMORY LAYOUT:
┌─────────────────────────────────────────────────────────────────────┐
│                    SHARED MEMORY REGION                              │
├─────────────────────────────────────────────────────────────────────┤
│ [General Metadata] <- Ipc related informations            												  │
├─────────────────────────────────────────────────────────────────────┤
│ [AllocatorMetadata] <- Allocator control structures              │
├─────────────────────────────────────────────────────────────────────┤
│ [BufferRegistry] <- Maps buffer names to locations                   │ 
├─────────────────────────────────────────────────────────────────────┤
│ ┌─[Buffer1: "events"]──────────────┐ <- Allocated by TLSF           │
│ │ [SharedBufferData][CircularData] │                                 │
│ └─────────────────────────────────┘                                 │
│ ┌─[Buffer2: "control"]─────────────┐                                 │  
│ │ [SharedBufferData][CircularData] │                                 │
│ └─────────────────────────────────┘                                 │
│ ┌─[Buffer3: "data_stream"]─────────┐                                 │
│ │ [SharedBufferData][CircularData] │                                 │
│ └─────────────────────────────────┘                                 │
├─────────────────────────────────────────────────────────────────────┤
│                    FREE SPACE (managed by TLSF)                      │
└─────────────────────────────────────────────────────────────────────┘
```

```bash
Shared Memory:
┌─[SharedQueueData<T>]──────────────────┐
│ head: AtomicUsize                        │
│ tail: AtomicUsize                        │ ← NEVER DROPPED
│ capacity: 1000                           │
│ [ring buffer data: T; 1000]           │
└─────────────────────────────────────────┘
           ↑              ↑              ↑
    [Process A]    [Process B]    [Process C]
    PersistentQueue PersistentQueue PersistentQueue
    (handle only)   (handle only)   (handle only)
         ↓              ↓              ↓
    Safe to drop   Safe to drop   Safe to drop
```

All things should be initiated **per process** with consistency so we won't lose information while retain the functionality across.

---

> Refer `Claude`

We have `BufferRegistry` contains `BufferRegistryEntry` to store the concrete offset for `Queue`.

```rust
#[repr(C)]
#[derive(Clone, Copy)]
struct BufferRegistryEntry {
    name: [u8; MAX_BUFFER_NAME_LEN],    // Buffer name (null-terminated)
    buffer_offset: usize,               // Offset from shared memory base
    buffer_size: usize,                 // Total buffer size
    created_by_pid: u32,                // Process that created it
    ref_count: AtomicU32,               // Number of processes using it
    is_active: AtomicU32,               // 1 = active, 0 = deleted
}

impl BufferRegistryEntry {
    fn new() -> Self {
        Self {
            name: [0; MAX_BUFFER_NAME_LEN],
            buffer_offset: 0,
            buffer_size: 0,
            created_by_pid: 0,
            ref_count: AtomicU32::new(0),
            is_active: AtomicU32::new(0),
        }
    }
    
    fn set_name(&mut self, name: &str) {}
    
    fn get_name(&self) -> String {}
}

/// Registry of all buffers - lives in shared memory right after TLSF metadata
#[repr(C, align(64))]
pub struct SharedBufferRegistry {
    magic: u32,
    entry_count: AtomicUsize,
    entries: [BufferRegistryEntry; MAX_BUFFERS],
}

impl SharedBufferRegistry {
    const MAGIC: u32 = 0x42554652; // "BUFR"
    
    pub fn initialize(&mut self) {
        self.magic = Self::MAGIC;
        self.entry_count.store(0, Ordering::Relaxed);
        
        // Initialize all entries
        for entry in &mut self.entries {
            *entry = BufferRegistryEntry::new();
        }
    }
    
    pub fn is_initialized(&self) -> bool {
        self.magic == Self::MAGIC
    }
    
    /// Register a new buffer
    pub fn register_buffer(&mut self, name: &str, offset: usize, size: usize, pid: u32) -> Result<(), BufferError> {
        // Check if buffer already exists
        if self.find_buffer(name).is_some() {
            return Err(BufferError::AlreadyExists);
        }
        
        // Find empty slot
        for entry in &mut self.entries {
            if entry.is_active.load(Ordering::Acquire) == 0 {
                entry.set_name(name);
                entry.buffer_offset = offset;
                entry.buffer_size = size;
                entry.created_by_pid = pid;
                entry.ref_count.store(1, Ordering::Relaxed);
                entry.is_active.store(1, Ordering::Release);
                
                self.entry_count.fetch_add(1, Ordering::Relaxed);
                return Ok(());
            }
        }
        
        Err(BufferError::RegistryFull)
    }
    
    /// Find existing buffer by name
    pub fn find_buffer(&self, name: &str) -> Option<(usize, usize)> {
        for entry in &self.entries {
            if entry.is_active.load(Ordering::Acquire) == 1 && entry.get_name() == name {
                entry.ref_count.fetch_add(1, Ordering::Relaxed);
                return Some((entry.buffer_offset, entry.buffer_size));
            }
        }
        None
    }
    
    /// List all active buffers
    pub fn list_buffers(&self) -> Vec<String> {
        let mut buffers = Vec::new();
        for entry in &self.entries {
            if entry.is_active.load(Ordering::Acquire) == 1 {
                buffers.push(entry.get_name());
            }
        }
        buffers
    }
}
```

---

The `CommunicationManager` should contains the layout of memory and memory base.

```rust
/// Master communication manager that coordinates allocator and buffers
pub struct CommunicationManager {
    // Process-specific allocator instance
    allocator: crate::PersistentTlsfAllocator,
    
    // Buffer registry (shared)
    registry: NonNull<SharedBufferRegistry>,
    
    // Process-specific buffer handles cache
    active_buffers: HashMap<String, BufferHandle>,
    
    // Process identification
    process_id: u32,
    shared_memory_base: *mut u8,
}
```

```rust
// shm-on data
pub struct LfQueue<T,N> {}

pub struct SQHandle {
    q: LfQueue,
    // other field...
}

#[repr(C, align(64))]
pub struct SharedQueueData<T> {
    // Queue metadata
    capacity: usize,
    element_size: usize,
    
    // Atomic indices for lock-free operation
    head: AtomicUsize,              // Consumer index
    tail: AtomicUsize,              // Producer index
    
    // Statistics and health tracking
    total_enqueues: AtomicUsize,
    total_dequeues: AtomicUsize,
    overruns: AtomicUsize,          // Failed enqueues due to full queue
    
    // Process tracking for debugging
    producer_pid: AtomicU32,        // Last producer PID
    consumer_pid: AtomicU32,        // Last consumer PID
    
    // Initialization state
    magic: u32,
    is_initialized: AtomicU32,
    
    // The actual ring buffer data follows this struct in memory
    // We don't store it as a field because it's variable-sized
    _phantom: PhantomData<T>,
}

impl<T> SharedQueueData<T> {
    const MAGIC: u32 = 0x51554555; // "QUEU"
    
    /// Initialize the shared queue data (called once)
    pub unsafe fn initialize(&mut self, capacity: usize) {
        self.magic = Self::MAGIC;
        self.capacity = capacity;
        self.element_size = size_of::<T>();
        
        // Initialize atomic indices
        self.head.store(0, Ordering::Relaxed);
        self.tail.store(0, Ordering::Relaxed);
        
        // Initialize statistics
        self.total_enqueues.store(0, Ordering::Relaxed);
        self.total_dequeues.store(0, Ordering::Relaxed);
        self.overruns.store(0, Ordering::Relaxed);
        
        self.producer_pid.store(0, Ordering::Relaxed);
        self.consumer_pid.store(0, Ordering::Relaxed);
        
        // Zero out the ring buffer data
        let data_ptr = self.data_ptr();
        core::ptr::write_bytes(data_ptr, 0, capacity * size_of::<T>());
        
        // Mark as initialized (release ordering ensures all writes above are visible)
        self.is_initialized.store(1, Ordering::Release);
    }
    
    /// Check if queue is properly initialized
    pub fn is_ready(&self) -> bool {
        self.magic == Self::MAGIC && 
        self.is_initialized.load(Ordering::Acquire) == 1
    }
    
    /// Get pointer to the ring buffer data
    pub fn data_ptr(&self) -> *mut T {
        unsafe {
            (self as *const _ as *mut u8)
                .add(size_of::<SharedQueueData<T>>())
                .cast::<T>()
        }
    }
    
    /// Calculate total size needed for this queue
    pub fn total_size(capacity: usize) -> usize {
        size_of::<SharedQueueData<T>>() + (capacity * size_of::<T>())
    }
    
    /// Get current queue length (approximate - may be stale)
    pub fn len(&self) -> usize {
        let tail = self.tail.load(Ordering::Acquire);
        let head = self.head.load(Ordering::Acquire);
        
        if tail >= head {
            tail - head
        } else {
            self.capacity - (head - tail)
        }
    }
    
    /// Check if queue is empty (approximate)
    pub fn is_empty(&self) -> bool {
        self.head.load(Ordering::Acquire) == self.tail.load(Ordering::Acquire)
    }
    
    /// Check if queue is full (approximate) 
    pub fn is_full(&self) -> bool {
        let tail = self.tail.load(Ordering::Acquire);
        let head = self.head.load(Ordering::Acquire);
        let next_tail = (tail + 1) % self.capacity;
        next_tail == head
    }
}

// CRITICAL: No Drop implementation for SharedQueueData
// This ensures the data persists even when process-local handles are dropped

//==============================================================================
// PROCESS-LOCAL QUEUE HANDLES (These CAN be dropped safely)
//==============================================================================

#[derive(Debug)]
pub enum QueueError {
    NotInitialized,
    Full,
    Empty,
    CorruptedData,
}

/// Process-local handle to a persistent queue
/// This is what gets created/dropped by user code - it's safe to drop
pub struct PersistentQueue<T> (
    NonNull<SharedQueueData<T>>,
    // Note: No ownership of the actual queue data
)

// NonNull<LfQueue<T,N>>
// ShmPtr<LfQueue<T,N>>
// Persist<LfQueue<T,N>>
```

```rust
use core::sync::atomic::{AtomicUsize, AtomicU32, Ordering};
use core::ptr::NonNull;
use core::mem::{size_of, align_of};
use std::collections::HashMap;

/// UNIFIED ALLOCATION INTERFACE DESIGN
/// 
/// The interface handles three key flows:
/// 1. USER → QUEUE: User sends data, allocator provides space, queue manages transfer
/// 2. QUEUE → USER: Queue notifies of data, allocator manages space, user receives
/// 3. ALLOCATOR ↔ QUEUE: Coordination for memory management and cleanup
///
/// Key Philosophy: 
/// - User sees simple send/receive interface
/// - Queue handles message routing and ordering  
/// - Allocator manages underlying memory lifecycle
/// - All three coordinate seamlessly for optimal performance

//==============================================================================
// UNIFIED COMMUNICATION INTERFACE
//==============================================================================

#[derive(Debug)]
pub enum CommunicationError {
    AllocationFailed,
    QueueFull,
    QueueEmpty,
    InvalidMessage,
    ChannelNotFound,
    CorruptedData,
    InsufficientSpace,
}

/// Main interface for queue-based communication with integrated allocation
pub struct UnifiedCommunicationSystem {
    /// TLSF allocator for memory management
    allocator: crate::PersistentTlsfAllocator,
    
    /// Message queues by channel name
    queues: HashMap<String, crate::PersistentQueue<MessageDescriptor>>,
    
    /// Queue factory for creating new queues
    queue_factory: crate::QueueFactory<crate::PersistentTlsfAllocator>,
    
    /// Process identification
    process_id: u32,
    sequence_counter: u64,
    
    /// Shared memory base for pointer calculations
    shared_memory_base: *mut u8,
    
    /// Statistics
    messages_sent: usize,
    messages_received: usize,
    bytes_allocated: usize,
}

impl UnifiedCommunicationSystem {
    /// Initialize the communication system
    pub unsafe fn new(
        shared_memory: *mut u8,
        total_size: usize,
        process_id: u32
    ) -> Result<Self, CommunicationError> {
        
        // Initialize TLSF allocator
        let allocator = crate::PersistentTlsfAllocator::create_or_attach(
            shared_memory,
            total_size,
            process_id
        ).map_err(|_| CommunicationError::AllocationFailed)?;
        
        let queue_factory = crate::QueueFactory::new(allocator.clone(), process_id);
        
        Ok(UnifiedCommunicationSystem {
            allocator,
            queues: HashMap::new(),
            queue_factory,
            process_id,
            sequence_counter: 1,
            shared_memory_base: shared_memory,
            messages_sent: 0,
            messages_received: 0,
            bytes_allocated: 0,
        })
    }
    
    //==========================================================================
    // HIGH-LEVEL USER INTERFACE
    //==========================================================================
    
    /// Send data to a channel (allocates + enqueues)
    /// This is the main user-facing send interface
    pub fn send<T: Clone>(&mut self, channel: &str, message_type: u32, data: &T) -> Result<(), CommunicationError> {
        // 1. Ensure channel exists
        self.ensure_channel_exists(channel)?;
        
        // 2. Allocate space for the data
        let data_size = size_of::<T>();
        let data_ptr = self.allocate_data(data_size)?;
        
        // 3. Write data to shared memory
        unsafe {
            core::ptr::write(data_ptr.cast::<T>().as_ptr(), data.clone());
        }
        
        // 4. Create message descriptor
        let data_offset = unsafe {
            data_ptr.as_ptr() as usize - self.shared_memory_base as usize
        };
        
        let descriptor = MessageDescriptor::new(
            data_offset,
            data_size,
            message_type,
            self.process_id,
            self.sequence_counter,
        );
        
        // 5. Enqueue the descriptor
        let queue = self.queues.get(channel).unwrap();
        queue.enqueue(descriptor).map_err(|_| CommunicationError::QueueFull)?;
        
        // 6. Update state
        self.sequence_counter += 1;
        self.messages_sent += 1;
        self.bytes_allocated += data_size;
        
        Ok(())
    }
    
    /// Send raw bytes to a channel
    pub fn send_bytes(&mut self, channel: &str, message_type: u32, data: &[u8]) -> Result<(), CommunicationError> {
        // 1. Ensure channel exists
        self.ensure_channel_exists(channel)?;
        
        // 2. Allocate space
        let data_ptr = self.allocate_data(data.len())?;
        
        // 3. Copy data
        unsafe {
            core::ptr::copy_nonoverlapping(
                data.as_ptr(),
                data_ptr.as_ptr(),
                data.len()
            );
        }
        
        // 4. Create and enqueue descriptor
        let data_offset = unsafe {
            data_ptr.as_ptr() as usize - self.shared_memory_base as usize
        };
        
        let descriptor = MessageDescriptor::new(
            data_offset,
            data.len(),
            message_type,
            self.process_id,
            self.sequence_counter,
        );
        
        let queue = self.queues.get(channel).unwrap();
        queue.enqueue(descriptor).map_err(|_| CommunicationError::QueueFull)?;
        
        self.sequence_counter += 1;
        self.messages_sent += 1;
        self.bytes_allocated += data.len();
        
        Ok(())
    }
    
    /// Receive data from a channel (dequeues + provides access)
    /// Returns a handle that automatically manages memory lifecycle
    pub fn receive(&mut self, channel: &str) -> Result<MessageHandle, CommunicationError> {
        let queue = self.queues.get(channel)
            .ok_or(CommunicationError::ChannelNotFound)?;
        
        // Dequeue message descriptor
        let descriptor = queue.dequeue()
            .map_err(|_| CommunicationError::QueueEmpty)?;
        
        self.messages_received += 1;
        
        // Create handle for accessing the data
        Ok(MessageHandle::new(descriptor, self.shared_memory_base, &mut self.allocator))
    }
    
    /// Try to receive without blocking
    pub fn try_receive(&mut self, channel: &str) -> Result<MessageHandle, CommunicationError> {
        self.receive(channel) // In this implementation, receive is already non-blocking
    }
    
    /// Send with zero-copy optimization (advanced interface)
    pub fn send_zero_copy(&mut self, channel: &str, message_type: u32, size: usize) 
        -> Result<ZeroCopyWriter, CommunicationError> {
        
        self.ensure_channel_exists(channel)?;
        
        // Allocate space but don't write yet
        let data_ptr = self.allocate_data(size)?;
        
        let data_offset = unsafe {
            data_ptr.as_ptr() as usize - self.shared_memory_base as usize
        };
        
        Ok(ZeroCopyWriter::new(
            channel.to_string(),
            data_ptr,
            data_offset,
            size,
            message_type,
            self.process_id,
            self.sequence_counter,
        ))
    }
    
    //==========================================================================
    // CHANNEL MANAGEMENT
    //==========================================================================
    
    /// Create or get existing channel
    pub fn create_channel(&mut self, name: &str, queue_capacity: usize) -> Result<(), CommunicationError> {
        if self.queues.contains_key(name) {
            return Ok(()); // Already exists
        }
        
        let queue = self.queue_factory.create_queue::<MessageDescriptor>(queue_capacity)
            .map_err(|_| CommunicationError::AllocationFailed)?;
        
        self.queues.insert(name.to_string(), queue);
        Ok(())
    }
    
    /// Connect to existing channel (created by another process)
    pub fn connect_channel(&mut self, name: &str, queue_ptr: NonNull<crate::SharedQueueData<MessageDescriptor>>) 
        -> Result<(), CommunicationError> {
        
        if self.queues.contains_key(name) {
            return Ok(());
        }
        
        let queue = self.queue_factory.connect_to_queue(queue_ptr)
            .map_err(|_| CommunicationError::AllocationFailed)?;
        
        self.queues.insert(name.to_string(), queue);
        Ok(())
    }
    
    /// List all available channels
    pub fn list_channels(&self) -> Vec<String> {
        self.queues.keys().cloned().collect()
    }
    
    //==========================================================================
    // INTERNAL HELPERS
    //==========================================================================
    
    fn ensure_channel_exists(&mut self, channel: &str) -> Result<(), CommunicationError> {
        if !self.queues.contains_key(channel) {
            // Create with default capacity
            self.create_channel(channel, 1000)?;
        }
        Ok(())
    }
    
    fn allocate_data(&mut self, size: usize) -> Result<NonNull<u8>, CommunicationError> {
        let layout = core::alloc::Layout::from_size_align(size, 8)
            .map_err(|_| CommunicationError::AllocationFailed)?;
        
        self.allocator.allocate(layout)
            .map_err(|_| CommunicationError::AllocationFailed)
    }
}


//==============================================================================
// MESSAGE TYPES AND DATA STRUCTURES
//==============================================================================
/// Message metadata that travels through queues
/// This is what gets enqueued/dequeued - contains reference to actual data
#[repr(C)]
#[derive(Clone, Copy, Debug)]
pub struct MessageDescriptor {
    /// Offset from shared memory base to actual data
    data_offset: usize,
    
    /// Size of the allocated data block
    data_size: usize,
    
    /// Message type for application-level routing
    message_type: u32,
    
    /// Source process ID
    source_pid: u32,
    
    /// Sequence number for ordering
    sequence: u64,
    
    /// Timestamp for timeouts and debugging
    timestamp: u64,
    
    /// Reference count - tracks how many readers need this data
    ref_count: AtomicU32,
    
    /// Checksum for data integrity
    checksum: u32,
}

impl MessageDescriptor {
    /// Create new message descriptor
    pub fn new(data_offset: usize, data_size: usize, message_type: u32, 
               source_pid: u32, sequence: u64) -> Self {
        Self {
            data_offset,
            data_size,
            message_type,
            source_pid,
            sequence,
            timestamp: current_timestamp(),
            ref_count: AtomicU32::new(1),
            checksum: 0, // Will be computed later
        }
    }
    
    /// Get pointer to actual data in shared memory
    pub unsafe fn data_ptr(&self, shared_memory_base: *mut u8) -> *mut u8 {
        shared_memory_base.add(self.data_offset)
    }
    
    /// Increment reference count (for multi-reader scenarios)
    pub fn add_ref(&self) -> u32 {
        self.ref_count.fetch_add(1, Ordering::AcqRel) + 1
    }
    
    /// Decrement reference count, returns true if should be deallocated
    pub fn release_ref(&self) -> bool {
        self.ref_count.fetch_sub(1, Ordering::AcqRel) == 1
    }
}

//==============================================================================
// MESSAGE HANDLE - MANAGES DATA LIFECYCLE
//==============================================================================

/// Handle for received messages - automatically manages memory lifecycle
pub struct MessageHandle {
    descriptor: MessageDescriptor,
    shared_memory_base: *mut u8,
    allocator: *mut crate::PersistentTlsfAllocator, // For cleanup on drop
}

impl MessageHandle {
    fn new(
        descriptor: MessageDescriptor,
        shared_memory_base: *mut u8,
        allocator: &mut crate::PersistentTlsfAllocator,
    ) -> Self {
        Self {
            descriptor,
            shared_memory_base,
            allocator: allocator as *mut _,
        }
    }
    
    /// Get message metadata
    pub fn metadata(&self) -> &MessageDescriptor {
        &self.descriptor
    }
    
    /// Get message type
    pub fn message_type(&self) -> u32 {
        self.descriptor.message_type
    }
    
    /// Get source process ID
    pub fn source_pid(&self) -> u32 {
        self.descriptor.source_pid
    }
    
    /// Get data size
    pub fn data_size(&self) -> usize {
        self.descriptor.data_size
    }
    
    /// Read data as specific type
    pub fn read_as<T>(&self) -> Result<&T, CommunicationError> {
        if size_of::<T>() != self.descriptor.data_size {
            return Err(CommunicationError::InvalidMessage);
        }
        
        unsafe {
            let data_ptr = self.descriptor.data_ptr(self.shared_memory_base);
            Ok(&*(data_ptr as *const T))
        }
    }
    
    /// Read data as byte slice
    pub fn read_bytes(&self) -> &[u8] {
        unsafe {
            let data_ptr = self.descriptor.data_ptr(self.shared_memory_base);
            core::slice::from_raw_parts(data_ptr, self.descriptor.data_size)
        }
    }
    
    /// Clone the data out (for owned access)
    pub fn clone_data<T: Clone>(&self) -> Result<T, CommunicationError> {
        Ok(self.read_as::<T>()?.clone())
    }
    
    /// Keep the message for longer (increments reference count)
    pub fn keep_alive(&self) -> MessageHandle {
        self.descriptor.add_ref();
        MessageHandle {
            descriptor: self.descriptor,
            shared_memory_base: self.shared_memory_base,
            allocator: self.allocator,
        }
    }
}

impl Drop for MessageHandle {
    fn drop(&mut self) {
        // Decrement reference count and deallocate if last reference
        if self.descriptor.release_ref() {
            unsafe {
                let data_ptr = NonNull::new_unchecked(
                    self.descriptor.data_ptr(self.shared_memory_base)
                );
                
                if let Some(allocator) = self.allocator.as_mut() {
                    let _ = allocator.deallocate(data_ptr);
                }
            }
        }
    }
}

//==============================================================================
// ZERO-COPY WRITER
//==============================================================================

/// Zero-copy writer for high-performance scenarios
pub struct ZeroCopyWriter {
    channel: String,
    data_ptr: NonNull<u8>,
    data_offset: usize,
    size: usize,
    message_type: u32,
    source_pid: u32,
    sequence: u64,
    committed: bool,
}

impl ZeroCopyWriter {
    fn new(
        channel: String,
        data_ptr: NonNull<u8>,
        data_offset: usize,
        size: usize,
        message_type: u32,
        source_pid: u32,
        sequence: u64,
    ) -> Self {
        Self {
            channel,
            data_ptr,
            data_offset,
            size,
            message_type,
            source_pid,
            sequence,
            committed: false,
        }
    }
    
    /// Get mutable slice for direct writing
    pub fn as_mut_slice(&mut self) -> &mut [u8] {
        unsafe {
            core::slice::from_raw_parts_mut(self.data_ptr.as_ptr(), self.size)
        }
    }
    
    /// Write typed data directly
    pub fn write<T>(&mut self, data: &T) -> Result<(), CommunicationError> {
        if size_of::<T>() != self.size {
            return Err(CommunicationError::InvalidMessage);
        }
        
        unsafe {
            core::ptr::write(self.data_ptr.as_ptr() as *mut T, *data);
        }
        
        Ok(())
    }
    
    /// Commit the write (enqueues the message)
    pub fn commit(mut self, comm_system: &mut UnifiedCommunicationSystem) -> Result<(), CommunicationError> {
        let descriptor = MessageDescriptor::new(
            self.data_offset,
            self.size,
            self.message_type,
            self.source_pid,
            self.sequence,
        );
        
        let queue = comm_system.queues.get(&self.channel)
            .ok_or(CommunicationError::ChannelNotFound)?;
        
        queue.enqueue(descriptor)
            .map_err(|_| CommunicationError::QueueFull)?;
        
        self.committed = true;
        comm_system.sequence_counter += 1;
        comm_system.messages_sent += 1;
        
        Ok(())
    }
}

impl Drop for ZeroCopyWriter {
    fn drop(&mut self) {
        if !self.committed {
            // TODO: Deallocate the reserved space since it wasn't used
            // This requires access to the allocator, which we'd need to store
        }
    }
}

//==============================================================================
// USAGE EXAMPLES
//==============================================================================

/// Example: Basic send/receive
pub fn example_basic_usage() -> Result<(), CommunicationError> {
    let shared_memory = unsafe { allocate_shared_memory(1024 * 1024) };
    
    // Initialize communication system
    let mut comm = unsafe {
        UnifiedCommunicationSystem::new(shared_memory, 1024 * 1024, std::process::id())?
    };
    
    // Create a channel
    comm.create_channel("events", 1000)?;
    
    // Send some data
    let event_data = "System started";
    comm.send_bytes("events", 1, event_data.as_bytes())?;
    
    // Send structured data
    #[derive(Clone)]
    struct StatusUpdate {
        cpu_usage: f32,
        memory_usage: f32,
        timestamp: u64,
    }
    
    let status = StatusUpdate {
        cpu_usage: 45.2,
        memory_usage: 78.1,
        timestamp: 1234567890,
    };
    
    comm.send("events", 2, &status)?;
    
    // Receive data
    let message = comm.receive("events")?;
    match message.message_type() {
        1 => {
            let text = String::from_utf8_lossy(message.read_bytes());
            println!("Received text: {}", text);
        },
        2 => {
            let status = message.read_as::<StatusUpdate>()?;
            println!("CPU: {}%, Memory: {}%", status.cpu_usage, status.memory_usage);
        },
        _ => println!("Unknown message type"),
    }
    
    Ok(())
}

/// Example: Zero-copy high-performance writing
pub fn example_zero_copy() -> Result<(), CommunicationError> {
    let shared_memory = unsafe { allocate_shared_memory(1024 * 1024) };
    let mut comm = unsafe {
        UnifiedCommunicationSystem::new(shared_memory, 1024 * 1024, std::process::id())?
    };
    
    comm.create_channel("bulk_data", 100)?;
    
    // Zero-copy write large data
    let mut writer = comm.send_zero_copy("bulk_data", 10, 4096)?;
    let buffer = writer.as_mut_slice();
    
    // Fill buffer directly (no intermediate copies)
    for (i, byte) in buffer.iter_mut().enumerate() {
        *byte = (i % 256) as u8;
    }
    
    // Commit to queue
    writer.commit(&mut comm)?;
    
    Ok(())
}

//==============================================================================
// UTILITY FUNCTIONS
//==============================================================================

unsafe fn allocate_shared_memory(size: usize) -> *mut u8 {
    // Platform-specific shared memory allocation
    std::alloc::alloc(std::alloc::Layout::from_size_align(size, 4096).unwrap())
}

fn current_timestamp() -> u64 {
    // Platform-specific timestamp
    0 // Placeholder
}
```

---

To design the initiation of shared memory, we should take a backend.

```rust
pub trait AddrSpec {
    type Addr: MemoryAddr;
    type Flags: Copy;
}

pub trait Map<S: AddrSpec>: Sized {
    type Config;
    type Error: core::fmt::Debug;

    fn map(
        self,
        start: Option<S::Addr>,
        size: usize,
        flags: S::Flags,
        cfg: Self::Config,
    ) -> Result<Area<S, Self>, Self::Error>;
    fn unmap(area: &mut Area<S, Self>) -> Result<(), Self::Error>;
}

pub struct Area<S: AddrSpec, M: Map<S>> {
    va_range: AddrRange<S::Addr>,
    flags: S::Flags,
    bk: M,
}
```

To adapt the connection logic, one should take **once at a time** for a connection instance with fixed resolver like `name`. Then 

```rust
pub trait ShmInit {
    fn try_claim() -> bool {}
}

pub struct Ipc<T: ShmInit + ...> {
    // Shared memory information
    area: Area<M>

    // Components
    ipc: T
}

impl<T: ShmInit> Ipc<T> {
        /// Initialize as server - this does the SINGLE mmap()
    fn initialize_server(&mut self, name: &str, size: usize) -> Result<(), Self::Error> {
        // 1. Create shared memory (SINGLE mmap call)
        let (shared_memory_base, actual_size) = create_shared_memory(name, size)?;
        
        // 2. Initialize the shared memory structures
        let server = Ipc::new(shared_memory_base, actual_size, name)?;
        
        Ok(())
    }
    
    /// Connect as client - NO mmap(), connects to server instead
    fn connect_client(&mut self, name: &str) -> Result<(), Self::Error> {
        // 1. Connect to server process (not shared memory directly)
        let server_connection = connect_to(name)?;

        // init client same...
        self.attach(sever_connection);
        
        Ok(())
    }

    pub fn attach(&mut self, name: &str, size: usize) -> Result<(), InitError> {
        // 2. Get coordinator (always at offset 0)
        let coord_ptr = shared_memory_base as *mut SharedMemoryCoordinator;
        let coord = unsafe { NonNull::new_unchecked(coordinator_ptr) };
        self.coord = Some(coord);
        
        let coord_ref = unsafe { coordinator.as_mut() };
        coord_ref.try_claim()?;

        // acquire possible local handle...
        
        Ok(())
    }
}
```



