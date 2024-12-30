# Базовый образ Python
FROM python:3.10-slim

# Устанавливаем FFmpeg и зависимости
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Создаем рабочую директорию
WORKDIR /app

# Устанавливаем Python-зависимости
COPY bot/requirements.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Копируем код приложения
COPY bot/ /app/

# Создаем папку для загрузок
RUN mkdir -p /app/downloads

# Указываем команду запуска
CMD ["python", "main.py"]
