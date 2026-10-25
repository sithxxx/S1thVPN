"""Админ-панель: страницы на сервере (Jinja2) + HTMX для автообновления."""
from pathlib import Path
from fastapi import APIRouter, Depends, Request
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates
from ..api.peers import PeerRow, list_peers

templates = Jinja2Templates(directory=Path(__file__).parent / "templates")
router = APIRouter(prefix="/admin", include_in_schema=False)

@router.get("", response_class=HTMLResponse)
async def dashboard(request: Request, rows: list[PeerRow] = Depends(list_peers)) -> HTMLResponse:
    return templates.TemplateResponse(request, "dashboard.html", {"rows": rows})

@router.get("/peers-table", response_class=HTMLResponse)
async def peers_table(request: Request, rows: list[PeerRow] = Depends(list_peers)) -> HTMLResponse:
    return templates.TemplateResponse(request, "_peers_table.html", {"rows": rows})