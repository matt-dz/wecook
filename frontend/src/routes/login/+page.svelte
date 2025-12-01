<script lang="ts">
	import Input from '$lib/components/input/Input.svelte';
	import Button from '$lib/components/button/Button.svelte';
	import fetch, { parseError } from '$lib/http';
	import { login } from '$lib/auth';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { HTTPError } from 'ky';

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
			goto(resolve('/'));
		} catch (e) {
			if (e instanceof HTTPError) {
				error = await parseError(e);
				return;
			}
			console.error('failed to login: ', e);
			alert('failed to login. check logs.');
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex h-screen w-full items-center justify-center">
	<form
		class="flex w-full max-w-[250px] flex-col items-center gap-2 rounded-lg border border-solid border-gray-400 p-4 shadow-lg"
		onsubmit={onSubmit}
	>
		<h1 class="text-2xl">WeCook Login</h1>
		<div class="flex w-full flex-col space-y-1">
			<Input
				disabled={loading}
				className="text-sm"
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
			<Input
				disabled={loading}
				className="text-sm"
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

		<Button className="from-blue-300 to-blue-200 border-blue-400" type="submit" disabled={loading}>
			<p class="text-sm font-medium">Login</p>
		</Button>
	</form>
</div>
