<script lang="ts">
	import { Input } from '$lib/components/ui/input/index.js';
	import Button from '$lib/components/button/Button.svelte';
	import fetch from '$lib/http';
	import { parseError } from '$lib/errors/api';
	import { login } from '$lib/auth';
	import { goto, invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { HTTPError } from 'ky';
	import { toast } from 'svelte-sonner';

	const placeholderError = 'placeholder';
	let loading = $state(false);
	let error: string = $state(placeholderError);
	let email: string = $state('');
	let password: string = $state('');

	async function onSubmit(event: SubmitEvent) {
		event.preventDefault();
		error = placeholderError;
		loading = true;

		try {
			await login(fetch, {
				email: email,
				password: password
			});
			await invalidateAll();
			goto(resolve('/home'));
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error(err.data);
					error = err.data.message;
					return;
				}
			}
			console.error('failed to login: ', e);
			toast.error('Failed to login.');
		} finally {
			loading = false;
		}
	}
</script>

<div class="absolute top-0 bottom-0 flex w-full items-center justify-center px-4">
	<form
		class="flex w-full max-w-[375px] flex-col gap-2 rounded-2xl border border-solid p-6 shadow-lg"
		onsubmit={onSubmit}
	>
		<div class="mb-2">
			<h1 class="text-left text-lg">Login</h1>
			<p class="text-sm text-gray-500">Don't have credentials? Ask the owner for an invitation.</p>
		</div>
		<div class="flex w-full flex-col space-y-1">
			<label for="email" class="text-sm">Email</label>
			<Input
				disabled={loading}
				class="text-sm"
				bind:value={email}
				id="email"
				name="email"
				type="email"
				placeholder="email"
				autocomplete="email"
				autocapitalize="none"
				autocorrect="off"
				spellcheck="false"
				inputmode="email"
				required
			/>
		</div>

		<div class="mt-1 flex w-full flex-col space-y-1">
			<label for="email" class="text-sm">Password</label>
			<Input
				disabled={loading}
				class="text-sm"
				bind:value={password}
				id="password"
				name="password"
				type="password"
				placeholder="password"
				autocomplete="current-password"
				autocapitalize="none"
				autocorrect="off"
				spellcheck="false"
				inputmode="text"
				required
			/>
		</div>

		<p
			class="text-center text-sm text-red-600"
			class:invisible={error === placeholderError}
			class:visible={error !== placeholderError}
		>
			{error}
		</p>

		<Button
			className="from-blue-300 to-blue-200 border-blue-400 hover:from-blue-200 hover:to-blue-100"
			type="submit"
			disabled={loading}
		>
			<p class="text-sm font-medium">Login</p>
		</Button>
	</form>
</div>
