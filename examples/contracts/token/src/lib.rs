// Simple ERC20-like token contract for Layer-1 blockchain
// Demonstrates WASM smart contract development

#![no_std]

use core::panic::PanicInfo;

// Host function imports
extern "C" {
    fn storage_get(key_ptr: *const u8, key_len: u32, value_ptr: *mut u8) -> u32;
    fn storage_set(key_ptr: *const u8, key_len: u32, value_ptr: *const u8, value_len: u32) -> u32;
    fn get_caller(out_ptr: *mut u8);
    fn emit_log(data_ptr: *const u8, data_len: u32);
}

// Panic handler
#[panic_handler]
fn panic(_info: &PanicInfo) -> ! {
    loop {}
}

// Memory allocation (simple bump allocator)
static mut HEAP: [u8; 65536] = [0; 65536];
static mut HEAP_POS: usize = 0;

#[no_mangle]
pub extern "C" fn alloc(size: usize) -> *mut u8 {
    unsafe {
        let ptr = HEAP.as_mut_ptr().add(HEAP_POS);
        HEAP_POS += size;
        ptr
    }
}

// Helper functions
fn read_u64(data: &[u8], offset: usize) -> u64 {
    u64::from_le_bytes([
        data[offset],
        data[offset + 1],
        data[offset + 2],
        data[offset + 3],
        data[offset + 4],
        data[offset + 5],
        data[offset + 6],
        data[offset + 7],
    ])
}

fn write_u64(data: &mut [u8], offset: usize, value: u64) {
    let bytes = value.to_le_bytes();
    data[offset..offset + 8].copy_from_slice(&bytes);
}

// Storage helpers
fn get_balance(address: &[u8; 20]) -> u64 {
    let mut key = [0u8; 28];
    key[0..8].copy_from_slice(b"balance:");
    key[8..28].copy_from_slice(address);
    
    let mut value = [0u8; 8];
    unsafe {
        let len = storage_get(key.as_ptr(), 28, value.as_mut_ptr());
        if len == 8 {
            read_u64(&value, 0)
        } else {
            0
        }
    }
}

fn set_balance(address: &[u8; 20], amount: u64) {
    let mut key = [0u8; 28];
    key[0..8].copy_from_slice(b"balance:");
    key[8..28].copy_from_slice(address);
    
    let mut value = [0u8; 8];
    write_u64(&mut value, 0, amount);
    
    unsafe {
        storage_set(key.as_ptr(), 28, value.as_ptr(), 8);
    }
}

fn get_total_supply() -> u64 {
    let key = b"total_supply";
    let mut value = [0u8; 8];
    unsafe {
        let len = storage_get(key.as_ptr(), key.len() as u32, value.as_mut_ptr());
        if len == 8 {
            read_u64(&value, 0)
        } else {
            0
        }
    }
}

fn set_total_supply(amount: u64) {
    let key = b"total_supply";
    let mut value = [0u8; 8];
    write_u64(&mut value, 0, amount);
    
    unsafe {
        storage_set(key.as_ptr(), key.len() as u32, value.as_ptr(), 8);
    }
}

// Contract functions

// Initialize contract with initial supply
fn initialize(initial_supply: u64) {
    let mut caller = [0u8; 20];
    unsafe {
        get_caller(caller.as_mut_ptr());
    }
    
    set_balance(&caller, initial_supply);
    set_total_supply(initial_supply);
}

// Transfer tokens
fn transfer(to: &[u8; 20], amount: u64) -> bool {
    let mut from = [0u8; 20];
    unsafe {
        get_caller(from.as_mut_ptr());
    }
    
    let from_balance = get_balance(&from);
    if from_balance < amount {
        return false;
    }
    
    let to_balance = get_balance(to);
    
    set_balance(&from, from_balance - amount);
    set_balance(to, to_balance + amount);
    
    // Emit transfer event
    let mut event = [0u8; 48];
    event[0..20].copy_from_slice(&from);
    event[20..40].copy_from_slice(to);
    write_u64(&mut event, 40, amount);
    
    unsafe {
        emit_log(event.as_ptr(), 48);
    }
    
    true
}

// Get balance
fn balance_of(address: &[u8; 20]) -> u64 {
    get_balance(address)
}

// Main entry point
#[no_mangle]
pub extern "C" fn main(input_ptr: *const u8, input_len: u32) -> u64 {
    if input_len == 0 {
        return 0;
    }
    
    let input = unsafe {
        core::slice::from_raw_parts(input_ptr, input_len as usize)
    };
    
    // First byte is the function selector
    let selector = input[0];
    
    match selector {
        0 => {
            // Initialize
            if input.len() >= 9 {
                let initial_supply = read_u64(input, 1);
                initialize(initial_supply);
            }
            0
        }
        1 => {
            // Transfer
            if input.len() >= 29 {
                let mut to = [0u8; 20];
                to.copy_from_slice(&input[1..21]);
                let amount = read_u64(input, 21);
                
                if transfer(&to, amount) {
                    1
                } else {
                    0
                }
            } else {
                0
            }
        }
        2 => {
            // Balance of
            if input.len() >= 21 {
                let mut address = [0u8; 20];
                address.copy_from_slice(&input[1..21]);
                balance_of(&address)
            } else {
                0
            }
        }
        3 => {
            // Total supply
            get_total_supply()
        }
        _ => 0,
    }
}

