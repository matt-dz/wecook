export function formatDuration(minutes: number): string {
	if (minutes < 60) {
		return plural(minutes, 'minute');
	}

	const hours = minutes / 60;
	if (hours < 24) {
		return plural(Math.floor(hours), 'hour');
	}

	const days = hours / 24;
	if (days < 7) {
		return plural(Math.floor(days), 'day');
	}

	const weeks = days / 7;
	if (weeks < 4) {
		return plural(Math.floor(weeks), 'week');
	}

	const months = days / 30; // approximate
	return plural(Math.floor(months), 'month');
}

function plural(value: number, unit: string): string {
	return value === 1 ? `${value} ${unit}` : `${value} ${unit}s`;
}
