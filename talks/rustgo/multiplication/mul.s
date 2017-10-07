TEXT ·MultiplyByTwo(SB), 0, $2048-16
    MOVQ n+0(FP), DI   // Load the argument before messing with SP
    MOVQ SP, BX        // Save SP in a callee-saved registry
    ADDQ $2048, SP     // Rollback SP to reuse this function's frame
    ANDQ $~15, SP      // Align the stack to 16-bytes

    MOVQ ·_multiply_two(SB), AX
	CALL AX

    MOVQ BX, SP        // Restore SP
    MOVQ AX, ret+8(FP) // Place the return value on the stack
    RET
