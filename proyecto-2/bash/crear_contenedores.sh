#!/bin/bash

echo "=== GENERANDO 10 CONTENEDORES ==="
echo "Timestamp: $(date)"

# Usar tus imágenes exactas
LIVIANA1="img_liviana:v1"
LIVIANA2="img_liviana2:v1"
PESADA_CPU="img_pesada_cpu:v1"
PESADA_RAM="img_pesada_ram:v1"

# Contenedores protegidos que NO se deben tocar
PROTECTED_CONTAINERS=("clever_black" "grafana-so1-202200214")

TIMESTAMP=$(date +%s)

# Verificar que Docker esté funcionando
if ! docker ps &> /dev/null; then
    echo "ERROR: Docker no está funcionando o no tienes permisos"
    exit 1
fi

# Verificar que las imágenes existan
echo "Verificando imágenes..."
for img in "$LIVIANA1" "$LIVIANA2" "$PESADA_CPU" "$PESADA_RAM"; do
    if ! docker image inspect "$img" &> /dev/null; then
        echo "ERROR: Imagen $img no encontrada. Construye las imágenes primero con:"
        echo "cd /home/pablo/proyecto-2 && ./build_images.sh"
        exit 1
    fi
done

echo "Verificando contenedores protegidos..."
for protected in "${PROTECTED_CONTAINERS[@]}"; do
    if docker ps --format "{{.Names}}" | grep -q "^${protected}$"; then
        echo "✓ Contenedor protegido activo: $protected"
    else
        echo "⚠ AVISO: Contenedor protegido no encontrado: $protected"
    fi
done

echo ""
echo "Creando contenedores..."

success_count=0
error_count=0

for i in {1..10}; do
    # Lógica de selección según el proyecto:
    # 1-3: Contenedores livianos alternando
    # 4-5: Contenedores pesados
    # 6-10: Aleatorio
    
    if [ $i -le 3 ]; then
        # Primeros 3: contenedores livianos alternando
        if [ $((i % 2)) -eq 1 ]; then
            IMAGE=$LIVIANA1
            TYPE="liviana1"
        else
            IMAGE=$LIVIANA2
            TYPE="liviana2"
        fi
    elif [ $i -le 5 ]; then
        # 4to y 5to: contenedores pesados
        if [ $i -eq 4 ]; then
            IMAGE=$PESADA_CPU
            TYPE="pesada_cpu"
        else
            IMAGE=$PESADA_RAM
            TYPE="pesada_ram"
        fi
    else
        # Resto: aleatorio entre todas
        IMAGES=($LIVIANA1 $LIVIANA2 $PESADA_CPU $PESADA_RAM)
        TYPES=("liviana1" "liviana2" "pesada_cpu" "pesada_ram")
        RAND_INDEX=$((RANDOM % 4))
        IMAGE=${IMAGES[$RAND_INDEX]}
        TYPE=${TYPES[$RAND_INDEX]}
    fi
    
    # Generar nombre único
    NAME="ctn_${TYPE}_${i}_${TIMESTAMP}"
    
    # Verificar que el nombre no esté en uso
    if docker ps -a --format "{{.Names}}" | grep -q "^${NAME}$"; then
        echo "⚠ Nombre de contenedor ya existe: $NAME, agregando sufijo"
        NAME="${NAME}_$(date +%N)"
    fi
    
    echo "Creando contenedor $i: $NAME ($TYPE)..."
    
    # Crear contenedor con límites apropiados según tipo
    case "$TYPE" in
        "pesada_cpu")
            # CPU intensivo: más CPU, menos memoria
            if docker run -d --rm --name "$NAME" --memory="256m" --cpus="1.5" "$IMAGE" &> /dev/null; then
                echo "✓ Contenedor $i: $NAME ($TYPE) lanzado exitosamente"
                ((success_count++))
            else
                echo "✗ Error creando contenedor $NAME"
                ((error_count++))
            fi
            ;;
        "pesada_ram")
            # RAM intensivo: más memoria, menos CPU
            if docker run -d --rm --name "$NAME" --memory="512m" --cpus="0.5" "$IMAGE" &> /dev/null; then
                echo "✓ Contenedor $i: $NAME ($TYPE) lanzado exitosamente"
                ((success_count++))
            else
                echo "✗ Error creando contenedor $NAME"
                ((error_count++))
            fi
            ;;
        "liviana1")
            # Liviano con shell script
            if docker run -d --rm --name "$NAME" --memory="64m" --cpus="0.3" "$IMAGE" &> /dev/null; then
                echo "✓ Contenedor $i: $NAME ($TYPE) lanzado exitosamente"
                ((success_count++))
            else
                echo "✗ Error creando contenedor $NAME"
                ((error_count++))
            fi
            ;;
        "liviana2")
            # Liviano con Node.js, exponer puerto
            PORT=$((3000 + i + RANDOM % 1000))  # Puerto más aleatorio para evitar conflictos
            if docker run -d --rm --name "$NAME" --memory="128m" --cpus="0.3" -p "${PORT}:3000" "$IMAGE" &> /dev/null; then
                echo "✓ Contenedor $i: $NAME ($TYPE) lanzado en puerto $PORT"
                ((success_count++))
            else
                echo "✗ Error creando contenedor $NAME"
                ((error_count++))
            fi
            ;;
    esac
    
    # Pequeña pausa para evitar problemas de concurrencia
    sleep 0.2
done

echo ""
echo "=== RESUMEN DE CREACIÓN ==="
echo "✓ Contenedores creados exitosamente: $success_count"
echo "✗ Contenedores con error: $error_count"
echo ""

# Mostrar estado actual (excluyendo contenedores protegidos del conteo)
echo "Contenedores activos (excluyendo protegidos):"
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | head -20

echo ""
echo "=== ESTADÍSTICAS DETALLADAS ==="
echo "Total contenedores del proyecto: $(docker ps --filter "name=ctn_" --format "{{.Names}}" | wc -l)"

# Contar por tipo
liviana1_count=$(docker ps --filter "name=ctn_liviana1" --format "{{.Names}}" | wc -l)
liviana2_count=$(docker ps --filter "name=ctn_liviana2" --format "{{.Names}}" | wc -l)
cpu_count=$(docker ps --filter "name=ctn_pesada_cpu" --format "{{.Names}}" | wc -l)
ram_count=$(docker ps --filter "name=ctn_pesada_ram" --format "{{.Names}}" | wc -l)

echo "- Livianos tipo 1 (shell): $liviana1_count"
echo "- Livianos tipo 2 (node): $liviana2_count"
echo "- CPU intensivos: $cpu_count"
echo "- RAM intensivos: $ram_count"

total_light=$((liviana1_count + liviana2_count))
total_heavy=$((cpu_count + ram_count))

echo ""
echo "Resumen por categoría:"
echo "- Total livianos: $total_light (límite: 3)"
echo "- Total pesados: $total_heavy (límite: 2)"

# Verificar contenedores protegidos
echo ""
echo "Contenedores protegidos:"
for protected in "${PROTECTED_CONTAINERS[@]}"; do
    if docker ps --format "{{.Names}}" | grep -q "^${protected}$"; then
        status=$(docker ps --filter "name=${protected}" --format "{{.Status}}")
        echo "✓ $protected - $status"
    else
        echo "✗ $protected - NO ENCONTRADO"
    fi
done

echo ""
echo "Script completado - $(date)"