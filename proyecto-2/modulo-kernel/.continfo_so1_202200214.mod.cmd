savedcmd_continfo_so1_202200214.mod := printf '%s\n'   continfo_so1_202200214.o | awk '!x[$$0]++ { print("./"$$0) }' > continfo_so1_202200214.mod
