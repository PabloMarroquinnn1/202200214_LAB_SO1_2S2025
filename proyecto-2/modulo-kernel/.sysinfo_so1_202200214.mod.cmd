savedcmd_sysinfo_so1_202200214.mod := printf '%s\n'   sysinfo_so1_202200214.o | awk '!x[$$0]++ { print("./"$$0) }' > sysinfo_so1_202200214.mod
