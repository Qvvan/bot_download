from aiogram import Router
from aiogram.filters import CommandStart
from aiogram.types import Message

router = Router()


@router.message(CommandStart())
async def process_start_command(message: Message):
    await message.answer("Привет! Я бот для скачивания видео и аудио с YouTube и Instagram.\n\n"
                         "Пожалуйста, отправь мне ссылку на видео")