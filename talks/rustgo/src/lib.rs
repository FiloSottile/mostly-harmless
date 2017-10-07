#![no_std]

#![feature(lang_items, compiler_builtins_lib, core_intrinsics)]
use core::intrinsics;
#[allow(private_no_mangle_fns)] #[no_mangle] // rust-lang/rust#38281
#[lang = "panic_fmt"] fn panic_fmt() -> ! { unsafe { intrinsics::abort() } }
#[lang = "eh_personality"] extern fn eh_personality() {}
extern crate compiler_builtins; // rust-lang/rust#43264
// extern crate rlibc;

#[no_mangle]
pub extern fn multiply_two(a: u64) -> u64 {
   return a * 2;
}
