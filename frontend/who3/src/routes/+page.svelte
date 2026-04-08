<script lang="ts">
	import { PUBLIC_API_ENDPOINT } from '$env/static/public';

	interface Player {
		steam_id: string;
		name: string;
		consecutive_count: number;
		server_id: string;
		last_updated: string;
		map_names: string[];
	}

	let serverId = $state('');
	let minCount = $state(3);
	let players: Player[] = $state([]);
	let loading = $state(false);
	let error = $state('');

	async function fetchPlayers() {
		loading = true;
		error = '';
		players = [];

		try {
			const params = new URLSearchParams();
			if (serverId.trim()) {
				params.set('server_id', serverId.trim());
			}
			params.set('min', String(minCount));

			const res = await fetch(`${PUBLIC_API_ENDPOINT}/players?${params}`);
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}: ${await res.text()}`);
			}

			const data: Player[] | null = await res.json();
			players = data ?? [];
		} catch (e: any) {
			error = e.message ?? 'Unknown error';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>who3 — Consecutive Game Count Tracker</title>
</svelte:head>

<main>
	<h1>who3</h1>
	<p class="subtitle">Consecutive game count tracker</p>

	<form onsubmit={(e) => { e.preventDefault(); fetchPlayers(); }}>
		<div class="controls">
			<label>
				Server ID
				<input type="text" bind:value={serverId} placeholder="e.g. 1" />
			</label>

			<label>
				Min Consecutive
				<input type="number" bind:value={minCount} min="1" placeholder="3"/>
			</label>

			<button type="submit" disabled={loading}>
				{loading ? 'Loading…' : 'Fetch Players'}
			</button>
		</div>
	</form>

	{#if error}
		<p class="error">{error}</p>
	{/if}

	{#if players.length > 0}
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Steam ID</th>
					<th>Server</th>
					<th>Count</th>
					<th>Last Game</th>
					<th>Maps</th>
				</tr>
			</thead>
			<tbody>
				{#each players as player}
					<tr>
						<td>{player.name}</td>
						<td class="mono">{player.steam_id}</td>
						<td class="mono">{player.server_id}</td>
						<td class="center">{player.consecutive_count}</td>
						<td class="mono">{new Date(player.last_updated).toLocaleString()}</td>
						<td>{player.map_names.join(', ')}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else if !loading && !error}
		<p class="empty">No players to display. Hit <strong>Fetch Players</strong> to query.</p>
	{/if}
</main>

<style>
	:global(body) {
		margin: 0;
		background: #0f1117;
		color: #e2e8f0;
	}

	main {
		max-width: 900px;
		margin: 2rem auto;
		padding: 0 1rem;
		font-family: system-ui, -apple-system, sans-serif;
	}

	h1 {
		margin-bottom: 0;
		color: #f8fafc;
		font-size: 2rem;
		letter-spacing: -0.5px;
	}

	.subtitle {
		margin-top: 0.25rem;
		color: #64748b;
	}

	.controls {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
		margin: 1.5rem 0;
	}

	label {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.875rem;
		font-weight: 600;
		color: #94a3b8;
	}

	input {
		padding: 0.4rem 0.6rem;
		background: #1e2330;
		border: 1px solid #2d3748;
		border-radius: 4px;
		font-size: 0.925rem;
		color: #e2e8f0;
		outline: none;
	}

	input:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
	}

	input[type='number'] {
		width: 80px;
	}

	button {
		padding: 0.5rem 1.2rem;
		background: #2563eb;
		color: #fff;
		border: none;
		border-radius: 4px;
		font-size: 0.925rem;
		cursor: pointer;
		transition: background 0.15s;
	}

	button:hover:not(:disabled) {
		background: #3b82f6;
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.error {
		color: #f87171;
		background: #1f1316;
		padding: 0.5rem 1rem;
		border-radius: 4px;
		border: 1px solid #7f1d1d;
	}

	.empty {
		color: #475569;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		margin-top: 1rem;
	}

	th,
	td {
		padding: 0.5rem 0.75rem;
		border: 1px solid #1e2330;
		text-align: left;
	}

	th {
		background: #161b27;
		font-size: 0.875rem;
		font-weight: 600;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	td {
		background: #131720;
	}

	.mono {
		font-family: monospace;
		font-size: 0.85rem;
		color: #7dd3fc;
	}

	.center {
		text-align: center;
	}

	tbody tr:hover td {
		background: #1a2035;
	}
</style>
