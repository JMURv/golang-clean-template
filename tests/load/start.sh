#!/bin/bash

# Конфигурация
BASE_URL="http://localhost:8080"
RATE="100/s"        # RPS
DURATION="30s"      # Длительность теста
ROOT_DIR="./tests/load"


if ! command -v vegeta &> /dev/null; then
    echo "Vegeta не установлен. Установите с помощью: go install github.com/tsenart/vegeta/v12@latest"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "Ошибка: Установите jq с помощью sudo apt-get install jq"
    exit 1
fi


echo "Создаем файл целей..."
cat > targets.txt <<EOF

POST $BASE_URL/api/
Authorization: Bearer $TOKEN
Content-Type: application/json
@sendCoin_body.json

EOF

echo "Запускаем нагрузочный тест..."
vegeta attack \
  -rate="$RATE" \
  -duration="$DURATION" \
  -targets=$ROOT_DIR/targets.txt \
  > $ROOT_DIR/results.bin

echo "Генерируем отчеты..."
vegeta report $ROOT_DIR/results.bin > $ROOT_DIR/report.txt

echo "Тестирование завершено!"
echo "Результаты:"
echo " - Текстовый отчет: report.txt"

rm -f $ROOT_DIR/targets.txt $ROOT_DIR/results.bin