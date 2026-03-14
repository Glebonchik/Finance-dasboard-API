# ML Categorization Service

gRPC сервис для автоматической категоризации транзакций на основе описания.

## Архитектура

```
Go API Server → gRPC → Python ML Service → BERT Model → Category
```

## Установка

### 1. Создать виртуальное окружение

```bash
python -m venv venv
venv\Scripts\activate  # Windows
source venv/bin/activate  # Linux/Mac
```

### 2. Установить зависимости

```bash
pip install -r requirements.txt
```

### 3. Сгенерировать Python код из .proto

```bash
python -m grpc_tools.protoc -I../proto --python_out=. --grpc_python_out=. ../proto/categorization.proto
```

### 4. Запустить сервер

```bash
python src/server.py
```

Сервер запустится на `localhost:50051`

## gRPC сервис

### Методы

#### Categorize

Определяет категорию для транзакции.

**Request:**
```protobuf
message CategorizeRequest {
  string description = 1;  // "ПЯТЁРОЧКА МАГАЗИН 2547"
  float amount = 2;        // 1500.00
  string currency = 3;     // "RUB"
}
```

**Response:**
```protobuf
message CategorizeResponse {
  int32 category_id = 1;    // 1
  string category_name = 2; // "Продукты"
  float confidence = 3;     // 0.8
}
```

## Планы

### MVP (сейчас)
- gRPC сервер
- Rule-based категоризация (ключевые слова)
- Интеграция с Go API

### Future
- BERT модель (DeepPavlov/rubert-base-cased)
- Fine-tuning на данных транзакций
- Confidence scoring
- Обучение на feedback пользователя

## Тестирование

```bash
# Тестирование через grpcurl
grpcurl -plaintext -d '{"description": "ПЯТЁРОЧКА МАГАЗИН", "amount": 1000, "currency": "RUB"}' \
  localhost:50051 ml.CategorizationService/Categorize
```
