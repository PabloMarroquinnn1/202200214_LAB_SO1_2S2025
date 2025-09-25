#!/bin/bash

echo "=== CONSTRUYENDO IMÁGENES DOCKER PARA PROYECTO SO1 ==="

cd dockerfiles

echo " Construyendo imagen liviana 1..."
cd liviana1
docker build -t img_liviana:v1 .
cd ..

echo " Construyendo imagen liviana 2..."
cd liviana2
docker build -t img_liviana2:v1 .
cd ..

echo " Construyendo imagen pesada CPU..."
cd pesada_cpu
docker build -t img_pesada_cpu:v1 .
cd ..

echo " Construyendo imagen pesada RAM..."
cd pesada_ram
docker build -t img_pesada_ram:v1 .
cd ..

echo " TODAS LAS IMÁGENES CONSTRUIDAS EXITOSAMENTE"
echo ""
echo " Imágenes disponibles:"
docker images | grep "img_"
