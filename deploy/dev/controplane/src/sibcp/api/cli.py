"""sibctl — консольная утилита администратора (работает напрямую с БД)."""
import asyncio 
import typer 

from .db import make_engine, make_sessionmaker 
from .models import AdminUser 
from .security import hash_password 
from .settings import Settings 

app = typer.Typer(help="Администрирование S1thVPN")

@app.command("create-admin")
def create_admin(username: str, password: str = typer.Option(..., prompt=True, hide_input=True, 
                                                             confirmation_prompt=True)) -> None:
    """Создать администратора панели."""
    async def go() -> None:
        engine = make_engine(Settings().database_url) 
        async with make_sessionmaker(engine)() as s:
            s.add(AdminUser(username=username, password_hash=hash_password(password)))
            await s.commit()
        await engine.dispose()
    asyncio.run(go())
    typer.echo(f"администратор {username} создан")
    
@app.command("genkey")
def genkey() -> None:
    """Сгенерировать пару ключей"""
    from .keys import generate_keypair 
    kp = generate_keypair()
    type.echo(f"private: {kp.private}\npublic: {kp.public}")
    
if __name__ == "__main__":
    app()