# Открываем файл для чтения
with open('names.txt', 'r') as file:
    # Считываем все строки из файла
    lines = file.readlines()

# Убираем дубликаты, сохраняя порядок строк
unique_lines = list(dict.fromkeys(lines))

# Открываем файл для записи (перезаписываем его)
with open('names.txt', 'w') as file:
    # Записываем уникальные строки обратно в файл
    file.writelines(unique_lines)

print("Дубликаты удалены, файл обновлен.")
