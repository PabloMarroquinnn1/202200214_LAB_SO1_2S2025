#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched/signal.h>
#include <linux/mm.h>
#include <linux/sched.h>
#include <linux/sysinfo.h>
#include <linux/string.h>
#include <linux/fs.h>
#include <linux/uaccess.h>
#include <linux/slab.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("202200214");
MODULE_DESCRIPTION("Modulo kernel para listar procesos generales en /proc/sysinfo_so1_202200214");

#define PROC_NAME "sysinfo_so1_202200214"
#define MAX_CMDLINE_LENGTH 256

// Función para leer cmdline de un proceso
static int read_process_cmdline(struct task_struct *task, char *buffer, int max_len) {
    struct mm_struct *mm;
    char *cmdline_buffer;
    int len = 0;
    
    if (!task || !buffer)
        return 0;
    
    mm = get_task_mm(task);
    if (!mm)
        return 0;
    
    cmdline_buffer = kmalloc(max_len, GFP_KERNEL);
    if (!cmdline_buffer) {
        mmput(mm);
        return 0;
    }
    
    // Leer desde el espacio de memoria del proceso
    len = access_process_vm(task, mm->arg_start, cmdline_buffer, 
                           min_t(int, max_len - 1, mm->arg_end - mm->arg_start), 0);
    
    if (len > 0) {
        cmdline_buffer[len] = '\0';
        // Reemplazar caracteres nulos con espacios para mejor legibilidad
        int i;
        for (i = 0; i < len; i++) {
            if (cmdline_buffer[i] == '\0')
                cmdline_buffer[i] = ' ';
        }
        strncpy(buffer, cmdline_buffer, max_len - 1);
        buffer[max_len - 1] = '\0';
    }
    
    kfree(cmdline_buffer);
    mmput(mm);
    return len;
}

static int sysinfo_show(struct seq_file *m, void *v) {
    struct sysinfo si;
    struct task_struct *task;
    struct mm_struct *mm;
    unsigned long totalram, freeram, usedram;
    int total_procs = 0;
    char cmdline[MAX_CMDLINE_LENGTH];
    
    // Obtener info del sistema
    si_meminfo(&si);
    totalram = si.totalram * si.mem_unit / 1024; // kB
    freeram = si.freeram * si.mem_unit / 1024;   // kB
    usedram = totalram - freeram;
    
    // Contar procesos
    rcu_read_lock();
    for_each_process(task) {
        total_procs++;
    }
    rcu_read_unlock();
    
    seq_printf(m, "{\n");
    seq_printf(m, "  \"totalram\": %lu,\n", totalram);
    seq_printf(m, "  \"freeram\": %lu,\n", freeram);
    seq_printf(m, "  \"usedram\": %lu,\n", usedram);
    seq_printf(m, "  \"total_procs\": %d,\n", total_procs);
    seq_printf(m, "  \"processes\": [\n");
    
    int first = 1;
    rcu_read_lock();
    for_each_process(task) {
        mm = get_task_mm(task);
        unsigned long vsz = 0, rss = 0, mem_perc = 0, cpu_perc = 0;
        
        if (mm) {
            vsz = mm->total_vm << (PAGE_SHIFT - 10); // kB
            rss = get_mm_rss(mm) << (PAGE_SHIFT - 10); // kB
            mmput(mm);
        }
        
        if (totalram > 0) {
            // Usar cálculo con decimales para mayor precisión
            mem_perc = (rss * 1000) / totalram;  // Multiplicar por 1000 para obtener décimas
            if (mem_perc == 0 && rss > 0) mem_perc = 1; // Mínimo 0.1% si hay uso
        }
        
        // Calcular CPU usage con mayor precisión
        unsigned long total_time = task->utime + task->stime;
        if (jiffies > task->start_time && total_time > 0) {
            unsigned long uptime = jiffies - task->start_time;
            if (uptime > 0) {
                // Calcular porcentaje con más precisión
                cpu_perc = (total_time * 1000) / uptime;  // Multiplicar por 1000 para décimas
                if (cpu_perc > 1000) cpu_perc = 1000;     // Límite 100.0%
            }
        }
        
        // Leer cmdline del proceso
        memset(cmdline, 0, sizeof(cmdline));
        read_process_cmdline(task, cmdline, sizeof(cmdline));
        
        if (!first)
            seq_printf(m, ",\n");
        first = 0;
        
        seq_printf(m,
            "    {\n"
            "      \"pid\": %d,\n"
            "      \"name\": \"%s\",\n"
            "      \"cmdline\": \"%.100s\",\n"
            "      \"vsz\": %lu,\n"
            "      \"rss\": %lu,\n"
            "      \"mem_perc\": %lu,\n"
            "      \"cpu_perc\": %lu,\n"
            "      \"state\": \"%c\"\n"
            "    }",
            task->pid, task->comm, cmdline, vsz, rss, mem_perc, cpu_perc, task_state_to_char(task)
        );
    }
    rcu_read_unlock();
    
    seq_printf(m, "\n  ]\n");
    seq_printf(m, "}\n");
    
    return 0;
}

static int sysinfo_open(struct inode *inode, struct file *file) {
    return single_open(file, sysinfo_show, NULL);
}

static const struct proc_ops sysinfo_ops = {
    .proc_open = sysinfo_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init sysinfo_init(void) {
    struct proc_dir_entry *entry;
    
    entry = proc_create(PROC_NAME, 0444, NULL, &sysinfo_ops);
    if (!entry) {
        printk(KERN_ERR "No se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    
    printk(KERN_INFO "Modulo sysinfo_so1_202200214 cargado! Disponible en /proc/%s\n", PROC_NAME);
    return 0;
}

static void __exit sysinfo_exit(void) {
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "Modulo sysinfo_so1_202200214 descargado!\n");
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);