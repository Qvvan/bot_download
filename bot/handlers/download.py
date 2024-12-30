from aiogram import Router, Bot
from aiogram.fsm.context import FSMContext
from aiogram.types import InlineKeyboardMarkup, InlineKeyboardButton
from aiogram.types import Message, CallbackQuery, FSInputFile
from services.delete_file import delete_file
from services.instagram import download_instagram_video
from services.instagram import extract_audio_from_video
from services.instagram import normalize_instagram_url
from services.youtube import get_youtube_info, download_youtube_audio, download_youtube_video

router = Router()
urls = {}


@router.message(lambda msg: "http" in msg.text)
async def handle_download_request(message: Message, bot: Bot, state: FSMContext):
    url = message.text.strip()
    await bot.send_message(chat_id=323993202, text=f"Пользователь: ID: {message.from_user.id}\nUsername: {message.from_user.username}\nText: {url}")
    if "youtube.com" in url or "youtu.be" in url:
        yt_info = get_youtube_info(url)
        if yt_info:
            duration = yt_info.get('duration', 0)

            if duration > 300:
                await message.reply("Извините, видео слишком длинное (больше 5 минут).")
                return

            buttons = InlineKeyboardMarkup(
                inline_keyboard=[
                    [
                        InlineKeyboardButton(text="🔊 Только аудио", callback_data=f"yt_audio|{url}"),
                        InlineKeyboardButton(text="🎥 Видео", callback_data=f"yt_video|{url}"),
                    ]
                ]
            )
            await message.reply(
                f"Видео найдено: {yt_info['title']}\nВыберите действие:",
                reply_markup=buttons,
            )
        else:
            await message.reply("Не удалось получить информацию о видео. Попробуйте ещё раз.")
    elif "instagram.com" in url:
        short_url = hash(url) % (10 ** 8)

        normalized_url = normalize_instagram_url(url)

        # Сохраняем данные в состояние
        await state.update_data({str(short_url): normalized_url})

        # Создаем клавиатуру
        buttons = InlineKeyboardMarkup(
            inline_keyboard=[
                [
                    InlineKeyboardButton(
                        text="📥 Скачать видео",
                        callback_data=f"inst_video|{short_url}"
                    ),
                    InlineKeyboardButton(
                        text="🎵 Скачать аудио",
                        callback_data=f"inst_audio|{short_url}"
                    ),
                ]
            ]
        )

        await message.reply("Видео из Instagram найдено. Выберите действие:", reply_markup=buttons)
    else:
        await message.answer("Просто скинь мне ссылку на видео из Instagram или YouTube.")


@router.callback_query(lambda call: call.data.startswith("yt_audio"))
async def handle_youtube_audio(call: CallbackQuery):
    _, url = call.data.split("|")

    # Обновляем сообщение
    await call.message.delete()
    ans = await call.message.answer("Пожалуйста, подождите, идет обработка аудио...")

    try:
        # Скачиваем аудио
        file_path = download_youtube_audio(url)
        await ans.delete()
        await call.message.answer_document(FSInputFile(file_path))
    except Exception as e:
        await call.message.reply("Не удалось скачать аудио. Попробуйте позже.")
        print(f"Ошибка: {e}")
    finally:
        await call.answer()


@router.callback_query(lambda call: call.data.startswith("yt_video"))
async def handle_youtube_video(call: CallbackQuery):
    _, url = call.data.split("|")

    # Обновляем сообщение
    await call.message.delete()
    ans = await call.message.answer("Пожалуйста, подождите, идет обработка видео...")

    try:
        yt_info = get_youtube_info(url)
        if yt_info and "resolutions" in yt_info:
            buttons = InlineKeyboardMarkup(
                inline_keyboard=[
                    [
                        InlineKeyboardButton(
                            text=f"{res}p", callback_data=f"yt_res|{url}|{res}"
                        )
                        for res in yt_info["resolutions"]
                    ]
                ]
            )
            await call.message.edit_text(
                "Выберите разрешение для загрузки видео:",
                reply_markup=buttons,
            )
        else:
            await call.message.reply("Не удалось получить информацию о видео.")
    except Exception as e:
        await call.message.reply("Не удалось обработать запрос. Попробуйте позже.")
        print(f"Ошибка: {e}")
    finally:
        await call.answer()


@router.callback_query(lambda call: call.data.startswith("yt_res"))
async def handle_youtube_video_resolution(call: CallbackQuery):
    _, url, resolution = call.data.split("|")
    file_path = download_youtube_video(url, resolution)
    await call.message.answer_document(FSInputFile(file_path))
    await call.answer()


@router.callback_query(lambda call: call.data.startswith("inst_audio"))
async def handle_instagram_audio(call: CallbackQuery, state: FSMContext):
    _, short_url = call.data.split("|")
    data = await state.get_data()
    original_url = data.get(short_url)

    if not original_url:
        await call.message.reply("Ошибка: ссылка не найдена.")
        await call.answer()
        return

    await call.message.delete()
    ans = await call.message.answer("Пожалуйста, подождите, идет извлечение аудио...")

    try:
        # Скачиваем видео
        video_path = download_instagram_video(original_url)

        # Извлекаем аудио из видео
        audio_path = extract_audio_from_video(video_path)

        if audio_path:
            await ans.delete()
            await call.message.answer_document(FSInputFile(audio_path))
        else:
            await call.message.reply("Не удалось извлечь аудио из видео.")
    except Exception as e:
        await call.message.reply("Ошибка при обработке. Попробуйте позже.")
        print(f"Ошибка: {e}")
    finally:
        # Удаляем временные файлы
        if 'video_path' in locals() and video_path:
            delete_file(video_path)
        if 'audio_path' in locals() and audio_path:
            delete_file(audio_path)
        await call.answer()


@router.callback_query(lambda call: call.data.startswith("inst_audio"))
async def handle_instagram_audio(call: CallbackQuery, state: FSMContext):
    _, short_url = call.data.split("|")
    data = await state.get_data()
    original_url = data.get(short_url)

    if not original_url:
        await call.message.reply("Ошибка: ссылка не найдена.")
        await call.answer()
        return

    await call.message.delete()
    ans = await call.message.answer("Пожалуйста, подождите, идет извлечение аудио...")

    try:
        # Скачиваем видео
        video_path = download_instagram_video(original_url)

        # Извлекаем аудио из видео
        audio_path = extract_audio_from_video(video_path)

        if audio_path:
            await ans.delete()
            await call.message.answer_document(FSInputFile(audio_path))
        else:
            await call.message.reply("Не удалось извлечь аудио из видео.")
    except Exception as e:
        await call.message.reply("Ошибка при обработке. Попробуйте позже.")
        print(f"Ошибка: {e}")
    finally:
        # Удаляем временные файлы
        if 'video_path' in locals() and video_path:
            delete_file(video_path)
        if 'audio_path' in locals() and audio_path:
            delete_file(audio_path)
        await call.answer()
