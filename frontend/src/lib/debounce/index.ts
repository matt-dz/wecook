export function debounce<Args extends unknown[], R>(
	fn: (...args: Args) => Promise<R> | R,
	delay: number
) {
	let timer: ReturnType<typeof setTimeout> | undefined;
	let pendingResolve: ((value: R) => void) | undefined;
	let pendingReject: ((reason?: unknown) => void) | undefined;

	return (...args: Args): Promise<R> => {
		if (timer) {
			clearTimeout(timer);
		}

		return new Promise<R>((resolve, reject) => {
			pendingResolve = resolve;
			pendingReject = reject;

			timer = setTimeout(async () => {
				timer = undefined;
				try {
					const result = await fn(...args);
					pendingResolve?.(result);
				} catch (err) {
					pendingReject?.(err);
				}
			}, delay);
		});
	};
}
