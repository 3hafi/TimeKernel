# NOTES

- Storage location defaults to `~/.trajectory/trajectory.db` and `~/.trajectory/config.json`.
- Imports are idempotent using unique keys (`events.external_id`, `logs.unique_key`).
- Reality Engine is deterministic and tied to observed deltas in logs/events.
- What-if planner emits conservative/balanced/aggressive variants and feasibility warnings.
