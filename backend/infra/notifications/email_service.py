import logging
import smtplib
from email.message import EmailMessage
from email.mime.image import MIMEImage
from urllib.parse import urljoin

from fastapi.templating import Jinja2Templates

from backend.infra.config import TEMPLATES, SMTPSettings, settings


logger = logging.getLogger(__name__)


class EmailService:
    def __init__(
        self,
        smtp_settings: SMTPSettings,
        templates: Jinja2Templates,
    ) -> None:
        self.smtp_settings = smtp_settings
        self.templates = templates

    @staticmethod
    def build_confirmation_link(
        name_endpoint: str,
        base_url: str,
        token: str,
    ) -> str:
        url = urljoin(base_url, f"/{name_endpoint}?token={token}")
        logger.info("Ссылка с токеном создана: %s", url)
        return url

    def compose_email(
        self,
        name_message: str,
        to_email: str,
        confirm_link: str,
    ) -> EmailMessage:
        message = EmailMessage()
        message["Subject"] = "Подтвердите почту"
        message["From"] = self.smtp_settings.user
        message["To"] = to_email

        text = f"Пожалуйста, подтвердите свою почту, перейдя по ссылке:\n{confirm_link}"
        html = self.templates.get_template(f"info/{name_message}.html").render(
            confirm_link=confirm_link
        )

        message.set_content(text)
        message.add_alternative(html, subtype="html")
        logger.info("Email-сообщение собрано.")
        return message

    def send_email(self, message: EmailMessage, to_email: str) -> bool:
        smtp_obj = None
        try:
            logger.info("Попытка подключения к SMTP серверу...")
            smtp_obj = smtplib.SMTP(self.smtp_settings.host, self.smtp_settings.port)
            smtp_obj.ehlo()
            if self.smtp_settings.use_tls:
                logger.debug("Инициализация TLS...")
                smtp_obj.starttls()
                smtp_obj.ehlo()
            smtp_obj.login(
                self.smtp_settings.user,
                self.smtp_settings.password.get_secret_value(),
            )
            smtp_obj.send_message(message)
            logger.info("Письмо успешно отправлено на %s", to_email)
            return True
        except Exception as e:
            logger.error("Ошибка при отправке письма: %s", e)
            return False
        finally:
            if smtp_obj:
                smtp_obj.quit()

    def statistics_message_configuration(self, statistic: bytes) -> EmailMessage:
        message = EmailMessage()
        message["Subject"] = "Статистика регистраций"
        message["From"] = self.smtp_settings.user
        message["To"] = str(self.smtp_settings.user_to_email or self.smtp_settings.user)

        text = "Статистика по зарегистрированным пользователям за последние сутки"
        message.set_content(text)

        image = MIMEImage(statistic, _subtype="png")
        image.add_header(
            "Content-Disposition",
            "attachment",
            filename="statistic.png",
        )
        message.add_attachment(
            image.get_payload(decode=True),
            maintype="image",
            subtype="png",
            filename="statistic.png",
        )

        logger.info("Email-сообщение статистики собрано.")
        return message

    def sending_a_message_from_statistic(self, message: EmailMessage) -> None:
        smtp_obj = None
        try:
            logger.info("Попытка подключения к SMTP серверу...")
            smtp_obj = smtplib.SMTP(self.smtp_settings.host, self.smtp_settings.port)
            smtp_obj.ehlo()
            if self.smtp_settings.use_tls:
                logger.debug("Инициализация TLS...")
                smtp_obj.starttls()
                smtp_obj.ehlo()
            smtp_obj.login(
                self.smtp_settings.user,
                self.smtp_settings.password.get_secret_value(),
            )
            smtp_obj.send_message(message)
            logger.info("Письмо статистики успешно отправлено")
        except Exception as e:
            logger.error("Ошибка при отправке письма: %s", e)
        finally:
            if smtp_obj:
                smtp_obj.quit()


def create_email_service(
    smtp_settings: SMTPSettings = settings.smtp,
    templates: Jinja2Templates = TEMPLATES,
) -> EmailService:
    return EmailService(
        smtp_settings=smtp_settings,
        templates=templates,
    )
