import { useState, useEffect, useCallback, useMemo } from 'preact/hooks';
import { GW_BASE } from '../api';
import { Modal } from '../components/Modal';
import { PlusIcon, RefreshIcon, TrashIcon, SearchIcon } from '../components/Icons';

interface CognitoBrowserProps {
  showToast: (msg: string) => void;
}

interface UserPool {
  Id: string;
  Name: string;
  CreationDate?: number;
  LastModifiedDate?: number;
}

interface CognitoUser {
  Username: string;
  Attributes: { Name: string; Value: string }[];
  UserCreateDate?: number;
  UserLastModifiedDate?: number;
  Enabled: boolean;
  UserStatus: string;
}

async function cognitoRequest(action: string, body: any) {
  const res = await fetch(GW_BASE, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-amz-json-1.1',
      'X-Amz-Target': `AWSCognitoIdentityProviderService.${action}`,
      'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260321/us-east-1/cognito-idp/aws4_request, SignedHeaders=host, Signature=fake',
    },
    body: JSON.stringify(body),
  });
  return res.json();
}

function getUserAttr(user: CognitoUser, name: string): string {
  return user.Attributes?.find(a => a.Name === name)?.Value || '';
}

function formatDate(ts: number | undefined): string {
  if (!ts) return '--';
  try {
    return new Date(ts * 1000).toLocaleString();
  } catch {
    return '--';
  }
}

function statusBadge(status: string) {
  const colors: Record<string, string> = {
    CONFIRMED: 'var(--success, #10b981)',
    UNCONFIRMED: 'var(--warning, #f59e0b)',
    FORCE_CHANGE_PASSWORD: 'var(--brand-blue, #097ff5)',
    COMPROMISED: 'var(--error, #ff4e5e)',
    RESET_REQUIRED: 'var(--warning, #f59e0b)',
  };
  const color = colors[status] || 'var(--text-tertiary)';
  return (
    <span style={{
      display: 'inline-block',
      padding: '2px 8px',
      borderRadius: '10px',
      fontSize: '11px',
      fontWeight: 600,
      color: 'white',
      background: color,
    }}>{status}</span>
  );
}

export function CognitoBrowserPage({ showToast }: CognitoBrowserProps) {
  const [pools, setPools] = useState<UserPool[]>([]);
  const [poolUserCounts, setPoolUserCounts] = useState<Record<string, number>>({});
  const [selectedPool, setSelectedPool] = useState<UserPool | null>(null);
  const [users, setUsers] = useState<CognitoUser[]>([]);
  const [poolSearch, setPoolSearch] = useState('');
  const [userSearch, setUserSearch] = useState('');
  const [loading, setLoading] = useState(false);

  // Modals
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [newEmail, setNewEmail] = useState('');
  const [newTempPassword, setNewTempPassword] = useState('');
  const [deleteUserConfirm, setDeleteUserConfirm] = useState<string | null>(null);
  const [confirmUserTarget, setConfirmUserTarget] = useState<string | null>(null);
  const [resetPasswordTarget, setResetPasswordTarget] = useState<string | null>(null);
  const [selectedUser, setSelectedUser] = useState<CognitoUser | null>(null);

  const loadPools = useCallback(async () => {
    try {
      const res = await cognitoRequest('ListUserPools', { MaxResults: 60 });
      const list: UserPool[] = res.UserPools || [];
      setPools(list);

      const counts: Record<string, number> = {};
      for (const pool of list) {
        try {
          const usersRes = await cognitoRequest('ListUsers', { UserPoolId: pool.Id, Limit: 1 });
          // Use estimated count from response or count returned users
          counts[pool.Id] = usersRes.Users?.length ?? 0;
          // Try to get a better count
          try {
            const desc = await cognitoRequest('DescribeUserPool', { UserPoolId: pool.Id });
            counts[pool.Id] = desc.UserPool?.EstimatedNumberOfUsers ?? counts[pool.Id];
          } catch { /* keep count from ListUsers */ }
        } catch {
          counts[pool.Id] = 0;
        }
      }
      setPoolUserCounts(counts);
    } catch {
      setPools([]);
    }
  }, []);

  useEffect(() => { loadPools(); }, [loadPools]);

  const loadUsers = useCallback(async (poolId: string) => {
    setLoading(true);
    try {
      const res = await cognitoRequest('ListUsers', { UserPoolId: poolId, Limit: 60 });
      setUsers(res.Users || []);
    } catch {
      setUsers([]);
      showToast('Failed to list users');
    }
    setLoading(false);
  }, [showToast]);

  function selectPool(pool: UserPool) {
    setSelectedPool(pool);
    setSelectedUser(null);
    setUserSearch('');
    loadUsers(pool.Id);
  }

  async function createUser() {
    if (!selectedPool || !newUsername.trim()) return;
    try {
      const params: any = {
        UserPoolId: selectedPool.Id,
        Username: newUsername.trim(),
        TemporaryPassword: newTempPassword || undefined,
        UserAttributes: [],
      };
      if (newEmail.trim()) {
        params.UserAttributes.push({ Name: 'email', Value: newEmail.trim() });
        params.UserAttributes.push({ Name: 'email_verified', Value: 'true' });
      }
      await cognitoRequest('AdminCreateUser', params);
      showToast(`User "${newUsername.trim()}" created`);
      setShowCreateUser(false);
      setNewUsername('');
      setNewEmail('');
      setNewTempPassword('');
      loadUsers(selectedPool.Id);
      loadPools();
    } catch (e: any) {
      showToast('Failed to create user');
    }
  }

  async function deleteUser(username: string) {
    if (!selectedPool) return;
    try {
      await cognitoRequest('AdminDeleteUser', {
        UserPoolId: selectedPool.Id,
        Username: username,
      });
      showToast(`User "${username}" deleted`);
      setDeleteUserConfirm(null);
      if (selectedUser?.Username === username) setSelectedUser(null);
      loadUsers(selectedPool.Id);
      loadPools();
    } catch {
      showToast('Failed to delete user');
      setDeleteUserConfirm(null);
    }
  }

  async function confirmUser(username: string) {
    if (!selectedPool) return;
    try {
      await cognitoRequest('AdminConfirmSignUp', {
        UserPoolId: selectedPool.Id,
        Username: username,
      });
      showToast(`User "${username}" confirmed`);
      setConfirmUserTarget(null);
      loadUsers(selectedPool.Id);
    } catch {
      showToast('Failed to confirm user');
      setConfirmUserTarget(null);
    }
  }

  async function resetPassword(username: string) {
    if (!selectedPool) return;
    try {
      await cognitoRequest('AdminResetUserPassword', {
        UserPoolId: selectedPool.Id,
        Username: username,
      });
      showToast(`Password reset for "${username}"`);
      setResetPasswordTarget(null);
      loadUsers(selectedPool.Id);
    } catch {
      showToast('Failed to reset password');
      setResetPasswordTarget(null);
    }
  }

  const displayPools = useMemo(() => {
    if (!poolSearch) return pools;
    const q = poolSearch.toLowerCase();
    return pools.filter(p => p.Name.toLowerCase().includes(q) || p.Id.toLowerCase().includes(q));
  }, [pools, poolSearch]);

  const displayUsers = useMemo(() => {
    if (!userSearch) return users;
    const q = userSearch.toLowerCase();
    return users.filter(u =>
      u.Username.toLowerCase().includes(q) ||
      getUserAttr(u, 'email').toLowerCase().includes(q)
    );
  }, [users, userSearch]);

  return (
    <div class="ddb-layout">
      {/* Pool sidebar */}
      <div class="ddb-sidebar">
        <div class="ddb-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">User Pools</span>
            <div class="flex gap-2">
              <button class="btn-icon btn-sm btn-ghost" title="Refresh" onClick={loadPools}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <RefreshIcon />
              </button>
            </div>
          </div>
          <div style="position:relative">
            <SearchIcon style="position:absolute;left:8px;top:50%;transform:translateY(-50%);color:var(--text-tertiary)" />
            <input class="input w-full" placeholder="Filter pools..." value={poolSearch}
              onInput={(e) => setPoolSearch((e.target as HTMLInputElement).value)}
              style="padding-left:30px;font-size:13px" />
          </div>
        </div>
        <div class="ddb-sidebar-list">
          {displayPools.map(pool => (
            <div key={pool.Id}
              class={`ddb-table-item ${selectedPool?.Id === pool.Id ? 'active' : ''}`}
              onClick={() => selectPool(pool)}>
              <div style="overflow:hidden">
                <div class="name" title={pool.Name}>{pool.Name}</div>
                <div style="font-size:10px;color:var(--text-tertiary);font-family:var(--font-mono);overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{pool.Id}</div>
              </div>
              <span class="count">{poolUserCounts[pool.Id] ?? '...'}</span>
            </div>
          ))}
          {displayPools.length === 0 && (
            <div style="padding:24px;text-align:center;font-size:13px;color:var(--text-tertiary)">
              No user pools found
            </div>
          )}
        </div>
      </div>

      {/* User manager */}
      <div class="ddb-main">
        {!selectedPool ? (
          <div class="empty-state">
            <div style="font-size:48px;opacity:0.3">Cognito</div>
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a user pool to manage users</div>
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">User pools are listed in the sidebar</div>
          </div>
        ) : selectedUser ? (
          /* User detail view */
          <div>
            <div style="margin-bottom:16px">
              <button class="btn btn-ghost btn-sm" onClick={() => setSelectedUser(null)}>
                Back to user list
              </button>
            </div>
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedUser.Username}</h2>
                <div style="display:flex;gap:8px;align-items:center">
                  {statusBadge(selectedUser.UserStatus)}
                  <span style="font-size:13px;color:var(--text-secondary)">
                    {selectedUser.Enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
              </div>
              <div class="flex gap-2">
                {selectedUser.UserStatus !== 'CONFIRMED' && (
                  <button class="btn btn-primary btn-sm" onClick={() => setConfirmUserTarget(selectedUser.Username)}>
                    Confirm User
                  </button>
                )}
                <button class="btn btn-ghost btn-sm" onClick={() => setResetPasswordTarget(selectedUser.Username)}>
                  Reset Password
                </button>
                <button class="btn btn-danger btn-sm" onClick={() => setDeleteUserConfirm(selectedUser.Username)}>
                  <TrashIcon /> Delete
                </button>
              </div>
            </div>

            <div class="card" style="margin-bottom:16px">
              <div class="card-header"><h3 style="font-weight:700">Attributes</h3></div>
              <div class="card-body">
                <table class="data-table" style="width:100%">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(selectedUser.Attributes || []).map(attr => (
                      <tr key={attr.Name}>
                        <td style="font-weight:600;font-size:13px">{attr.Name}</td>
                        <td style="font-family:var(--font-mono);font-size:12px">{attr.Value}</td>
                      </tr>
                    ))}
                    {(!selectedUser.Attributes || selectedUser.Attributes.length === 0) && (
                      <tr>
                        <td colSpan={2} style="text-align:center;color:var(--text-tertiary);padding:16px">No attributes</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            <div class="card">
              <div class="card-header"><h3 style="font-weight:700">Details</h3></div>
              <div class="card-body">
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;font-size:13px">
                  <div>
                    <div class="label">Created</div>
                    <div>{formatDate(selectedUser.UserCreateDate)}</div>
                  </div>
                  <div>
                    <div class="label">Last Modified</div>
                    <div>{formatDate(selectedUser.UserLastModifiedDate)}</div>
                  </div>
                  <div>
                    <div class="label">Status</div>
                    <div>{selectedUser.UserStatus}</div>
                  </div>
                  <div>
                    <div class="label">Enabled</div>
                    <div>{selectedUser.Enabled ? 'Yes' : 'No'}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        ) : (
          /* User list */
          <div>
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedPool.Name}</h2>
                <div style="font-size:13px;color:var(--text-secondary);font-family:var(--font-mono)">{selectedPool.Id}</div>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-primary btn-sm" onClick={() => setShowCreateUser(true)}>
                  <PlusIcon /> Create User
                </button>
                <button class="btn btn-ghost btn-sm" onClick={() => loadUsers(selectedPool.Id)}>
                  <RefreshIcon /> Refresh
                </button>
              </div>
            </div>

            <div style="margin-bottom:12px;position:relative">
              <SearchIcon style="position:absolute;left:8px;top:50%;transform:translateY(-50%);color:var(--text-tertiary)" />
              <input class="input w-full" placeholder="Filter users..." value={userSearch}
                onInput={(e) => setUserSearch((e.target as HTMLInputElement).value)}
                style="padding-left:30px;font-size:13px" />
            </div>

            {loading ? (
              <div style="padding:32px;text-align:center;color:var(--text-tertiary)">Loading...</div>
            ) : (
              <table class="data-table" style="width:100%">
                <thead>
                  <tr>
                    <th>Username</th>
                    <th>Email</th>
                    <th>Status</th>
                    <th>Created</th>
                    <th style="width:160px">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {displayUsers.map(user => (
                    <tr key={user.Username}>
                      <td>
                        <span style="cursor:pointer;color:var(--brand-blue);font-weight:600"
                          onClick={() => setSelectedUser(user)}>
                          {user.Username}
                        </span>
                      </td>
                      <td style="font-size:13px">{getUserAttr(user, 'email') || '--'}</td>
                      <td>{statusBadge(user.UserStatus)}</td>
                      <td style="font-size:12px;color:var(--text-secondary)">{formatDate(user.UserCreateDate)}</td>
                      <td>
                        <div class="flex gap-2">
                          {user.UserStatus !== 'CONFIRMED' && (
                            <button class="btn btn-ghost btn-sm" style="padding:2px 8px;font-size:11px"
                              onClick={() => setConfirmUserTarget(user.Username)}>
                              Confirm
                            </button>
                          )}
                          <button class="btn btn-ghost btn-sm" style="padding:2px 8px;font-size:11px"
                            onClick={() => setResetPasswordTarget(user.Username)}>
                            Reset
                          </button>
                          <button class="btn btn-danger btn-sm" style="padding:2px 6px;font-size:11px"
                            onClick={() => setDeleteUserConfirm(user.Username)}>
                            <TrashIcon />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {displayUsers.length === 0 && (
                    <tr>
                      <td colSpan={5} style="text-align:center;padding:24px;color:var(--text-tertiary)">
                        No users in this pool
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>

      {/* Create User Modal */}
      {showCreateUser && (
        <Modal title="Create User" size="sm" onClose={() => setShowCreateUser(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setShowCreateUser(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={createUser}>Create</button>
            </>
          }>
          <div class="mb-4">
            <div class="label">Username</div>
            <input class="input w-full" placeholder="john.doe" value={newUsername}
              onInput={(e) => setNewUsername((e.target as HTMLInputElement).value)} />
          </div>
          <div class="mb-4">
            <div class="label">Email</div>
            <input class="input w-full" type="email" placeholder="john@example.com" value={newEmail}
              onInput={(e) => setNewEmail((e.target as HTMLInputElement).value)} />
          </div>
          <div class="mb-4">
            <div class="label">Temporary Password</div>
            <input class="input w-full" type="password" placeholder="TempPass123!" value={newTempPassword}
              onInput={(e) => setNewTempPassword((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => { if (e.key === 'Enter') createUser(); }} />
          </div>
        </Modal>
      )}

      {/* Delete User Confirm */}
      {deleteUserConfirm && (
        <Modal title="Delete User" size="sm" onClose={() => setDeleteUserConfirm(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setDeleteUserConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={() => deleteUser(deleteUserConfirm)}>Delete</button>
            </>
          }>
          <p>Are you sure you want to delete user <strong>{deleteUserConfirm}</strong>? This cannot be undone.</p>
        </Modal>
      )}

      {/* Confirm User */}
      {confirmUserTarget && (
        <Modal title="Confirm User" size="sm" onClose={() => setConfirmUserTarget(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setConfirmUserTarget(null)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={() => confirmUser(confirmUserTarget)}>Confirm</button>
            </>
          }>
          <p>Confirm user <strong>{confirmUserTarget}</strong>? This will mark them as verified.</p>
        </Modal>
      )}

      {/* Reset Password */}
      {resetPasswordTarget && (
        <Modal title="Reset Password" size="sm" onClose={() => setResetPasswordTarget(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setResetPasswordTarget(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={() => resetPassword(resetPasswordTarget)}>Reset</button>
            </>
          }>
          <p>Reset password for <strong>{resetPasswordTarget}</strong>? They will need to set a new password.</p>
        </Modal>
      )}
    </div>
  );
}
