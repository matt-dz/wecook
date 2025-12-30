import type { ColumnDef } from '@tanstack/table-core';
import type { User } from '$lib/users';
import { renderComponent } from '../ui/data-table';
import DatatableActions from './table-actions.svelte';
import EmailCell from './email-cell.svelte';

export const getColumns = (
	onUserDeleted: (userId: number) => void,
	currentUserId: number,
	totalUsers: number
): ColumnDef<User>[] => [
	{
		accessorKey: 'role',
		header: 'Role'
	},
	{
		accessorKey: 'email',
		header: 'Email',
		cell: ({ row }) => {
			return renderComponent(EmailCell, {
				email: row.original.email,
				isCurrentUser: row.original.id === currentUserId
			});
		}
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
			const isCurrentUser = row.original.id === currentUserId;
			const isOnlyUser = totalUsers === 1;

			return renderComponent(DatatableActions, {
				id: row.original.id,
				email: row.original.email,
				onUserDeleted,
				isCurrentUser,
				canDelete: !(isCurrentUser && isOnlyUser)
			});
		}
	}
];
