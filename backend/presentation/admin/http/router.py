from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter

from backend.application.use_cases.admin import AdminRbacUseCase
from backend.domain.user.entity import User
from backend.presentation.shared.http.schemas import AdminAuditLogResponse, AdminRoleResponse, MessageResponse


router = APIRouter(prefix="/admin", tags=["Admin"], route_class=DishkaRoute)


@router.get("/roles", response_model=list[AdminRoleResponse])
async def list_roles(use_case: FromDishka[AdminRbacUseCase], current_user: FromDishka[User]) -> list[AdminRoleResponse]:
    return [AdminRoleResponse.model_validate(role, from_attributes=True) for role in await use_case.list_roles(current_user)]


@router.post("/users/{user_id:int}/roles/{role_name}", response_model=MessageResponse)
async def assign_role(user_id: int, role_name: str, use_case: FromDishka[AdminRbacUseCase], current_user: FromDishka[User]) -> MessageResponse:
    return MessageResponse(message=(await use_case.assign_role(current_user, user_id, role_name)).message)


@router.delete("/users/{user_id:int}/roles/{role_name}", response_model=MessageResponse)
async def remove_role(user_id: int, role_name: str, use_case: FromDishka[AdminRbacUseCase], current_user: FromDishka[User]) -> MessageResponse:
    return MessageResponse(message=(await use_case.remove_role(current_user, user_id, role_name)).message)


@router.get("/audit", response_model=list[AdminAuditLogResponse])
async def list_audit(
    use_case: FromDishka[AdminRbacUseCase],
    current_user: FromDishka[User],
    limit: int = 50,
) -> list[AdminAuditLogResponse]:
    logs = await use_case.list_audit(current_user, limit)
    return [AdminAuditLogResponse.model_validate(log, from_attributes=True) for log in logs]
