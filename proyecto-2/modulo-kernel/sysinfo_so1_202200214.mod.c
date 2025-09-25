#include <linux/module.h>
#include <linux/export-internal.h>
#include <linux/compiler.h>

MODULE_INFO(name, KBUILD_MODNAME);

__visible struct module __this_module
__section(".gnu.linkonce.this_module") = {
	.name = KBUILD_MODNAME,
	.init = init_module,
#ifdef CONFIG_MODULE_UNLOAD
	.exit = cleanup_module,
#endif
	.arch = MODULE_ARCH_INIT,
};



static const struct modversion_info ____versions[]
__used __section("__versions") = {
	{ 0x5218fe90, "single_open" },
	{ 0xfefac423, "remove_proc_entry" },
	{ 0xc7ffe1aa, "si_meminfo" },
	{ 0xd272d446, "__rcu_read_lock" },
	{ 0x43f4e0dd, "init_task" },
	{ 0xd272d446, "__rcu_read_unlock" },
	{ 0x12cfb334, "seq_printf" },
	{ 0xf3a689eb, "get_task_mm" },
	{ 0x98ba4f31, "mmput" },
	{ 0x058c185a, "jiffies" },
	{ 0xbd03ed67, "random_kmalloc_seed" },
	{ 0xc2fefbb5, "kmalloc_caches" },
	{ 0x38395bf3, "__kmalloc_cache_noprof" },
	{ 0x7a7dfafe, "access_process_vm" },
	{ 0xc609ff70, "strncpy" },
	{ 0xcb8b6ec6, "kfree" },
	{ 0x90a48d82, "__ubsan_handle_out_of_bounds" },
	{ 0xd272d446, "__stack_chk_fail" },
	{ 0xd22cd56f, "seq_read" },
	{ 0x388dee05, "seq_lseek" },
	{ 0xae030cd0, "single_release" },
	{ 0xd272d446, "__fentry__" },
	{ 0xf8d7ac5e, "proc_create" },
	{ 0xe8213e80, "_printk" },
	{ 0xd272d446, "__x86_return_thunk" },
	{ 0x70eca2ca, "module_layout" },
};

static const u32 ____version_ext_crcs[]
__used __section("__version_ext_crcs") = {
	0x5218fe90,
	0xfefac423,
	0xc7ffe1aa,
	0xd272d446,
	0x43f4e0dd,
	0xd272d446,
	0x12cfb334,
	0xf3a689eb,
	0x98ba4f31,
	0x058c185a,
	0xbd03ed67,
	0xc2fefbb5,
	0x38395bf3,
	0x7a7dfafe,
	0xc609ff70,
	0xcb8b6ec6,
	0x90a48d82,
	0xd272d446,
	0xd22cd56f,
	0x388dee05,
	0xae030cd0,
	0xd272d446,
	0xf8d7ac5e,
	0xe8213e80,
	0xd272d446,
	0x70eca2ca,
};
static const char ____version_ext_names[]
__used __section("__version_ext_names") =
	"single_open\0"
	"remove_proc_entry\0"
	"si_meminfo\0"
	"__rcu_read_lock\0"
	"init_task\0"
	"__rcu_read_unlock\0"
	"seq_printf\0"
	"get_task_mm\0"
	"mmput\0"
	"jiffies\0"
	"random_kmalloc_seed\0"
	"kmalloc_caches\0"
	"__kmalloc_cache_noprof\0"
	"access_process_vm\0"
	"strncpy\0"
	"kfree\0"
	"__ubsan_handle_out_of_bounds\0"
	"__stack_chk_fail\0"
	"seq_read\0"
	"seq_lseek\0"
	"single_release\0"
	"__fentry__\0"
	"proc_create\0"
	"_printk\0"
	"__x86_return_thunk\0"
	"module_layout\0"
;

MODULE_INFO(depends, "");


MODULE_INFO(srcversion, "3B6BE9FB06D6CC1CB0A7334");
