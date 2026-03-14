"""
ML Categorization Service gRPC Server

Заглушка для демонстрации gRPC интеграции.
В будущем будет заменена на полноценную BERT модель.
"""

import grpc
from concurrent import futures
import categorization_pb2
import categorization_pb2_grpc

# Заглушка: простые правила для категоризации по ключевым словам
CATEGORY_KEYWORDS = {
    1: ("Продукты", ["пятерочка", "магнит", "перекресток", "спар", "азбука", "продукты", "еда"]),
    2: ("Транспорт", ["метро", "автобус", "такси", "uber", "yandex", "транспорт", "бензин", "лукойл"]),
    3: ("Рестораны", ["ресторан", "кафе", "бар", "макдоналдс", "бургер", "кофе", "старбакс"]),
    4: ("Здоровье", ["аптека", "больница", "клиника", "врач", "лекарство", "здоровье"]),
    5: ("Развлечения", ["кино", "театр", "концерт", "игра", "развлечение", "парк"]),
    6: ("Дом", ["мебель", "ремонт", "хозтовары", "дом", "квартира"]),
    7: ("Одежда", ["одежда", "обувь", "магазин", "мода", "zara", "h&m"]),
    8: ("Красота", ["салон", "косметика", "красота", "парикмахер"]),
    9: ("Образование", ["книги", "курсы", "образование", "школа", "университет"]),
    10: ("Переводы", ["перевод", "transfer", "сбербанк", "тинькофф"]),
    11: ("Налоги и сборы", ["налог", "штраф", "пошлина", "госуслуги"]),
    12: ("Доходы", ["зарплата", "доход", "перевод", "возврат"]),
    13: ("Другое", []),
}


class CategorizationServicer(categorization_pb2_grpc.CategorizationServiceServicer):
    """Реализация gRPC сервиса категоризации"""

    def Categorize(self, request, context):
        """
        Определяет категорию для транзакции на основе описания.
        
        Для MVP используется простое правило ключевых слов.
        В будущем будет заменена на BERT модель.
        """
        description = request.description.lower()
        amount = request.amount
        currency = request.currency

        # Поиск по ключевым словам
        for category_id, (category_name, keywords) in CATEGORY_KEYWORDS.items():
            for keyword in keywords:
                if keyword in description:
                    return categorization_pb2.CategorizeResponse(
                        category_id=category_id,
                        category_name=category_name,
                        confidence=0.8  # Уверенность для rule-based подхода
                    )

        # Категория по умолчанию
        return categorization_pb2.CategorizeResponse(
            category_id=13,  # Другое
            category_name="Другое",
            confidence=0.3  # Низкая уверенность
        )


def serve():
    """Запуск gRPC сервера"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    categorization_pb2_grpc.add_CategorizationServiceServicer_to_server(
        CategorizationServicer(), server
    )
    
    port = "50051"
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    
    print(f"ML gRPC server started on port {port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
