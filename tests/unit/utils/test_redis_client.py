async def test_save_and_get_token(redis_test_client):
    token = "token123"
    email = "user@example.com"

    # Тестируем сохранение токена
    result = await redis_test_client.save_registration_confirmation_token(token, email)
    assert result is True

    # Тестируем получение токена
    value = await redis_test_client.get_email_by_token(token)
    assert value == email

    # Проверяем существование токена
    assert value == email

    # Удаляем токен
    await redis_test_client.delete_token(token)

    # После удаления токен не должен существовать
    value_after_delete = await redis_test_client.get_email_by_token(token)
    assert value_after_delete is None
