import os
import subprocess

import requests
from instaloader import Instaloader, Post

def download_instagram_video(url: str) -> str:
    """
    Скачивает видео из Instagram.

    :param url: Ссылка на Instagram пост.
    :return: Путь к загруженному видео.
    """
    loader = Instaloader()
    shortcode = url.split("/")[-2]
    post = Post.from_shortcode(loader.context, shortcode)

    if not post.is_video:
        raise ValueError("Этот пост не содержит видео.")

    # Создаем папку для загрузок
    os.makedirs("downloads", exist_ok=True)

    # Скачиваем видео
    video_path = f"downloads/{post.owner_username}_{post.shortcode}.mp4"
    response = requests.get(post.video_url, stream=True)
    with open(video_path, "wb") as video_file:
        for chunk in response.iter_content(chunk_size=8192):
            video_file.write(chunk)

    return video_path


def extract_audio_from_video(video_path: str) -> str:
    """
    Извлекает аудио из видео с помощью FFmpeg.

    :param video_path: Путь к видеофайлу.
    :return: Путь к аудиофайлу.
    """
    audio_path = video_path.replace(".mp4", ".mp3")

    # Команда FFmpeg для извлечения аудио
    command = [
        "ffmpeg", "-i", video_path, "-q:a", "0", "-map", "a", audio_path, "-y"
    ]

    try:
        subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
        return audio_path
    except subprocess.CalledProcessError as e:
        print(f"Ошибка при извлечении аудио: {e}")
        return None