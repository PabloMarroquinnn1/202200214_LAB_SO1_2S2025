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
MODULE_DESCRIPTION("Modulo kernel para listar procesos de contenedores en /proc/continfo_so1_202200214");

#define PROC_NAME "continfo_so1_202200214"
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

// Función mejorada para determinar si un proceso es de Docker/contenedor
static int is_container_process(struct task_struct *task) {
    char cmdline[MAX_CMDLINE_LENGTH];
    int len;
    
    if (!task)
        return 0;
    
    // Leer cmdline primero
    len = read_process_cmdline(task, cmdline, sizeof(cmdline));
    
    // Solo buscar procesos que específicamente corresponden a nuestros contenedores
    if (len > 0) {
        // SOLO procesos de nuestros contenedores específicos
        if (strstr(cmdline, "/cpu_stress.py") ||
            strstr(cmdline, "/ram_stress.py") ||
            strstr(cmdline, "/app.js") ||
            strstr(cmdline, "/app.sh") ||
            strstr(cmdline, "python3 /cpu_stress.py") ||
            strstr(cmdline, "python3 /ram_stress.py") ||
            strstr(cmdline, "node /app.js") ||
            strstr(cmdline, "sh /app.sh")) {
            return 1;
        }
    }
    
    return 0;  // Ser más restrictivo
}

// Función para extraer ID de contenedor de cmdline
static void extract_container_id(char *cmdline, char *container_id, int max_len, int pid) {
    char *ptr;
    int i, j = 0;
    
    // Inicializar con PID por defecto
    snprintf(container_id, max_len, "container_%d", pid);
    
    // Buscar patrones comunes de ID de contenedor
    if ((ptr = strstr(cmdline, "ctn_")) != NULL) {
        // Extraer nombre del contenedor que empiece con ctn_
        ptr += 4; // Saltar "ctn_"
        j = 0;
        for (i = 0; i < max_len - 1 && ptr[i] && ptr[i] != ' ' && ptr[i] != '\n'; i++) {
            container_id[j++] = ptr[i];
        }
        container_id[j] = '\0';
    } else if ((ptr = strstr(cmdline, "--name")) != NULL) {
        // Buscar parámetro --name
        ptr = strchr(ptr, ' ');
        if (ptr) {
            ptr++; // Saltar espacio
            j = 0;
            for (i = 0; i < max_len - 1 && ptr[i] && ptr[i] != ' ' && ptr[i] != '\n'; i++) {
                container_id[j++] = ptr[i];
            }
            container_id[j] = '\0';
        }
    } else if (strstr(cmdline, "stress")) {
        // Para contenedores de stress
        snprintf(container_id, max_len, "stress_container_%d", pid);
    } else if (strstr(cmdline, "python3") && strstr(cmdline, "stress")) {
        // Para contenedores Python de stress
        snprintf(container_id, max_len, "python_stress_%d", pid);
    }
}

static int continfo_show(struct seq_file *m, void *v) {
    struct sysinfo si;
    struct task_struct *task;
    struct mm_struct *mm;
    unsigned long totalram, freeram, usedram;
    int container_count = 0;
    char cmdline[MAX_CMDLINE_LENGTH];
    char container_id[64];
    
    // Obtener info del sistema
    si_meminfo(&si);
    totalram = si.totalram * si.mem_unit / 1024; // kB
    freeram = si.freeram * si.mem_unit / 1024;   // kB
    usedram = totalram - freeram;
    
    // Contar contenedores primero
    rcu_read_lock();
    for_each_process(task) {
        if (is_container_process(task)) {
            container_count++;
        }
    }
    rcu_read_unlock();
    
    seq_printf(m, "{\n");
    seq_printf(m, "  \"totalram\": %lu,\n", totalram);
    seq_printf(m, "  \"freeram\": %lu,\n", freeram);
    seq_printf(m, "  \"usedram\": %lu,\n", usedram);
    seq_printf(m, "  \"total_containers\": %d,\n", container_count);
    seq_printf(m, "  \"containers\": [\n");
    
    int first = 1;
    rcu_read_lock();
    for_each_process(task) {
        if (is_container_process(task)) {
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
            
            // Extraer ID de contenedor
            extract_container_id(cmdline, container_id, sizeof(container_id), task->pid);
            
            if (!first)
                seq_printf(m, ",\n");
            first = 0;
            
            seq_printf(m,
                "    {\n"
                "      \"pid\": %d,\n"
                "      \"name\": \"%s\",\n"
                "      \"cmdline\": \"%.100s\",\n"
                "      \"container_id\": \"%s\",\n"
                "      \"vsz\": %lu,\n"
                "      \"rss\": %lu,\n"
                "      \"mem_perc\": %lu,\n"
                "      \"cpu_perc\": %lu\n"
                "    }",
                task->pid, task->comm, cmdline, container_id, vsz, rss, mem_perc, cpu_perc
            );
        }
    }
    rcu_read_unlock();
    
    seq_printf(m, "\n  ]\n");
    seq_printf(m, "}\n");
    
    return 0;
}

static int continfo_open(struct inode *inode, struct file *file) {
    return single_open(file, continfo_show, NULL);
}

static const struct proc_ops continfo_ops = {
    .proc_open = continfo_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init continfo_init(void) {
    struct proc_dir_entry *entry;
    
    entry = proc_create(PROC_NAME, 0444, NULL, &continfo_ops);
    if (!entry) {
        printk(KERN_ERR "No se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    
    printk(KERN_INFO "Modulo continfo_so1_202200214 cargado! Disponible en /proc/%s\n", PROC_NAME);
    return 0;
}

static void __exit continfo_exit(void) {
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "Modulo continfo_so1_202200214 descargado!\n");
}

module_init(continfo_init);
module_exit(continfo_exit);