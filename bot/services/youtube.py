import yt_dlp


def get_youtube_info(url: str) -> dict:
    """
    Получает информацию о видео на YouTube.

    :param url: Ссылка на YouTube-видео.
    :return: Словарь с информацией о видео (название, разрешения, продолжительность).
    """
    try:
        ydl_opts = {
            'format': 'bestvideo+bestaudio/best',
            'noplaylist': True,
        }
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info_dict = ydl.extract_info(url, download=False)
            title = info_dict.get('title', None)
            duration = info_dict.get('duration', 0)  # Продолжительность в секундах
            formats = info_dict.get('formats', [])

            # Получаем список разрешений
            resolutions = sorted(
                set(fmt['height'] for fmt in formats if fmt.get('height'))
            )

            print("Доступные разрешения:", resolutions)
            return {"title": title, "resolutions": resolutions, "duration": duration}
    except Exception as e:
        print(f"Error fetching YouTube info: {e}")
        return None



def download_youtube_audio(url: str) -> str:
    """
    Скачивает только аудио с YouTube.

    :param url: Ссылка на YouTube-видео.
    :return: Путь к скачанному файлу.
    """
    try:
        ydl_opts = {
            'format': 'bestaudio/best',  # Выбираем лучший доступный аудио формат
            'outtmpl': 'downloads/%(title)s.%(ext)s',  # Путь сохранения файла
            'postprocessors': [{  # Конвертация аудио в mp3
                'key': 'FFmpegExtractAudio',
                'preferredcodec': 'mp3',
                'preferredquality': '192',  # Качество аудио
            }],
        }
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(url, download=True)  # Скачиваем аудио
            return ydl.prepare_filename(info).replace('.webm', '.mp3')  # Возвращаем путь к файлу
    except Exception as e:
        print(f"Error downloading audio: {e}")
        return None


def download_youtube_video(url: str, resolution: str = "720") -> str:
    """
    Скачивает видео с YouTube в указанном разрешении.

    :param url: Ссылка на YouTube-видео.
    :param resolution: Разрешение видео (например, "720").
    :return: Путь к скачанному файлу.
    """
    try:
        ydl_opts = {
            'format': f'bestvideo[height={resolution}]+bestaudio/best',  # Формат: лучшее видео + лучшее аудио
            'outtmpl': 'downloads/%(title)s.%(ext)s',  # Путь сохранения файла
            'merge_output_format': 'mp4',  # Объединение видео и аудио в MP4
        }
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(url, download=True)  # Скачиваем видео
            return ydl.prepare_filename(info)  # Возвращаем путь к файлу
    except Exception as e:
        print(f"Error downloading video: {e}")
        return None
