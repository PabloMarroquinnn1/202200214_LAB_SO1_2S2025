obj-m += continfo_so1_202200214.o
obj-m += sysinfo_so1_202200214.o

all:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules

clean:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean

install:
	sudo insmod continfo_so1_202200214.ko
	sudo insmod sysinfo_so1_202200214.ko

uninstall:
	sudo rmmod continfo_so1_202200214
	sudo rmmod sysinfo_so1_202200214

test:
	cat /proc/continfo_so1_202200214
	echo "---"
	cat /proc/sysinfo_so1_202200214