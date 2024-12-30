import os

def delete_file(file_path: str):
    """
    Удаляет файл, если он существует.
    :param file_path: Путь к файлу.
    """
    try:
        if os.path.exists(file_path):
            os.remove(file_path)
            print(f"Файл удален: {file_path}")
        else:
            print(f"Файл не найден для удаления: {file_path}")
    except Exception as e:
        print(f"Ошибка при удалении файла: {e}")
