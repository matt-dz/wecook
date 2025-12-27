import type { ColumnDef } from '@tanstack/table-core';
import type { User } from '$lib/users';
import { renderComponent } from '../ui/data-table';
import DatatableActions from './table-actions.svelte';

export const columns: ColumnDef<User>[] = [
	{
		accessorKey: 'role',
		header: 'Role'
	},
	{
		accessorKey: 'email',
		header: 'Email'
	},
	{
		accessorKey: 'first_name',
		header: 'First Name'
	},
	{
		accessorKey: 'last_name',
		header: 'Last Name'
	},
	{
		id: 'actions',
		cell: ({ row }) => {
			return renderComponent(DatatableActions, { id: row.original.id.toString() });
		}
	}
];
