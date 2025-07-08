.globl f
f:
	pushq %%rax
	movq $3, %rax
	movq %%rax, %%rcx
	popq %%rax
	imulq %%rcx, %%rax
	ret
