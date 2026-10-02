from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse
from sqlalchemy import text

router = APIRouter(tags=["health"])

@router.get("/healthz/")
async def healthZ(request: Request) -> JSONResponse:
    try:
        async with request.app.state.sessionmaker() as s:
            await s.execute(text("SELECT 1"))
        return JSONResponse({"db": True})
    except Exception:
        return JSONResponse({"db": False}, status_code=503)