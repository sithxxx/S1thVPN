from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse 
from sqlalchemy import text

router = APIRouter(tags=["health"])

@router.get("/healthz")
async def healthz(request: Request) -> JSONResponse:
    checks = {"db": False, "dataplane": False}
    try:
        async with request.app.state.sessionmaker() as s:
            await s.execute(text("SELECT 1"))
        checks["db"] = True
    except Exception:
        pass
    try:
        await request.app.state.driver.ping()
        checks["dataplane"] = True
    except Exception:
        pass
    return JSONResponse(checks, status_code=200 if all(checks.values()) else 503)