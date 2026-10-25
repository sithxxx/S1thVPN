from prometheus_client import Counter, Gauge, make_asgi_app

PEERS_DESIRED = Gauge("sibcp_peers_desired", "Устройств, которые должны быть в датаплейне")
RECONCILE_RUNS = Counter("sibcp_reconcile_runs_total", "Циклов reconciler")
RECONCILE_ERRORS = Counter("sibcp_reconcile_errors_total", "Неудачных циклов reconciler")
RECONCILE_CHANGES = Counter("sibcp_reconcile_changes_total", "Изменений в датаплейне", ["kind"])
metrics_app = make_asgi_app()