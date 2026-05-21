"""Dishka container creation."""

from dishka import AsyncContainer, make_async_container
from dishka.integrations.fastapi import FastapiProvider

from backend.di.auth_provider import AuthProvider
from backend.di.providers import ApplicationProvider, InfrastructureProvider


def create_container() -> AsyncContainer:
    return make_async_container(
        FastapiProvider(),
        InfrastructureProvider(),
        AuthProvider(),
        ApplicationProvider(),
    )
