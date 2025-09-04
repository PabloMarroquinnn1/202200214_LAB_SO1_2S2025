#!/bin/bash

echo " Iniciando la simulación de creación de contenedores!"
echo " Ubicación actual del script:"
pwd
echo ""

echo " Generando número aleatorio de contenedores (entre 1 y 4)..."
num_contenedores=$((RANDOM % 4 + 1))
echo " Vamos a crear $num_contenedores contenedores"
echo ""

echo " Iniciando proceso de creación..."

for i in $(seq 1 $num_contenedores)
do
    echo "    Creando contenedor #$i..."
    
    nombre_aleatorio=""
    for j in {1..8}
    do
        caracteres="abcdefghijklmnopqrstuvwxyz0123456789"
        indice=$((RANDOM % ${#caracteres}))
        caracter=${caracteres:$indice:1}
        nombre_aleatorio="$nombre_aleatorio$caracter"
    done
    
    nombre_archivo="contenedor_202200214_$nombre_aleatorio.txt"
    echo "   Creando archivo: $nombre_archivo"
    
    touch "$nombre_archivo"
    echo "$nombre_archivo" > "$nombre_archivo"
    echo "   Contenedor $nombre_archivo creado exitosamente"
done

echo ""
echo "¡Proceso completado!"
echo "Resumen de contenedores creados:"
echo "Archivos creados en esta ejecución:"
ls -la contenedor_202200214_*.txt 2>/dev/null || echo "   ⚠️  No se encontraron archivos"

echo ""
echo "Contenido de los archivos creados:"
for archivo in contenedor_202200214_*.txt
do
    if [ -f "$archivo" ]; then
        echo "   Contenido de $archivo:"
        echo "      $(cat "$archivo")"
    fi
done

echo ""
echo " ¡Simulación de contenedores completada con éxito!"
echo " Los contenedores fueron creados en: $(pwd)"
