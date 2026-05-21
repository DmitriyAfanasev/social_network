import pytest
from backend.shared.utils.case_converter import camel_case_to_snake_case


@pytest.mark.parametrize(
    "input_str, expected",
    [
        # Базовые случаи
        ("simpleCase", "simple_case"),
        ("already_snake", "already_snake"),
        ("Single", "single"),
        # Аббревиатуры и сложные случаи
        ("SomeSDK", "some_sdk"),
        ("RServoDrive", "r_servo_drive"),
        ("SDKDemo", "sdk_demo"),
        ("JSONParser", "json_parser"),
        ("HTMLElement", "html_element"),
        ("CSSParser", "css_parser"),
        # Крайние случаи
        ("", ""),
        ("A", "a"),
        # Числа в строках
        ("model3D", "model3_d"),
        ("item2DArray", "item2_d_array"),
    ],
)
def test_camel_case_to_snake_case(input_str, expected):
    """Тестирует преобразование camelCase в snake_case с различными вариантами ввода."""
    assert camel_case_to_snake_case(input_str) == expected


def test_camel_case_to_snake_case_edge_cases():
    """Тестирует крайние случаи и специальные символы."""
    # Проверка сохранения подчеркиваний
    assert camel_case_to_snake_case("preserve__underscores") == "preserve__underscores"

    # Специальные символы
    assert camel_case_to_snake_case("special$Chars") == "special$_chars"
    assert camel_case_to_snake_case("special@Chars") == "special@_chars"
    assert camel_case_to_snake_case("with spaces") == "with spaces"

    # Числовые строки
    assert camel_case_to_snake_case("123") == "123"
    assert camel_case_to_snake_case("42Answer") == "42_answer"
