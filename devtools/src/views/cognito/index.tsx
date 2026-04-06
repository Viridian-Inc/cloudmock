import { useState, useEffect, useCallback, useMemo } from 'preact/hooks';
import { getAdminBase } from '../../lib/api';
import './cognito.css';

// ---------- Types ----------

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

// ---------- Helpers ----------

function gwBase(): string {
  return getAdminBase().replace(':4599', ':4566');
}

async function cognitoRequest(action: string, body: any) {
  const res = await fetch(gwBase(), {
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

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'CONFIRMED': return 'confirmed';
    case 'UNCONFIRMED': return 'unconfirmed';
    case 'FORCE_CHANGE_PASSWORD': return 'force-change';
    case 'COMPROMISED': return 'compromised';
    case 'RESET_REQUIRED': return 'reset-required';
    default: return 'default';
  }
}

// ---------- Inline Icons ----------

function SearchIcon(props: any) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14" {...props}>
      <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  );
}

function RefreshIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" />
      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <polyline points="3 6 5 6 21 6" /><path d="M19 6l-1.5 14a2 2 0 0 1-2 2h-7a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6" /><path d="M14 11v6" /><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  );
}

function XIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  );
}

function ArrowLeftIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <line x1="19" y1="12" x2="5" y2="12" /><polyline points="12 19 5 12 12 5" />
    </svg>
  );
}

// ---------- Modal ----------

interface ModalProps {
  title: string;
  onClose: () => void;
  footer?: preact.ComponentChildren;
  children: preact.ComponentChildren;
}

function Modal({ title, onClose, footer, children }: ModalProps) {
  return (
    <div class="cognito-modal-backdrop" onClick={onClose}>
      <div class="cognito-modal" onClick={(e) => e.stopPropagation()}>
        <div class="cognito-modal-header">
          <h3>{title}</h3>
          <button class="btn-icon" onClick={onClose}><XIcon /></button>
        </div>
        <div class="cognito-modal-body">{children}</div>
        {footer && <div class="cognito-modal-footer">{footer}</div>}
      </div>
    </div>
  );
}

// ---------- Main Component ----------

export function CognitoView() {
  const [pools, setPools] = useState<UserPool[]>([]);
  const [poolUserCounts, setPoolUserCounts] = useState<Record<string, number>>({});
  const [selectedPool, setSelectedPool] = useState<UserPool | null>(null);
  const [users, setUsers] = useState<CognitoUser[]>([]);
  const [poolSearch, setPoolSearch] = useState('');
  const [userSearch, setUserSearch] = useState('');
  const [loading, setLoading] = useState(false);

  // Modals
  const [showCreatePool, setShowCreatePool] = useState(false);
  const [newPoolName, setNewPoolName] = useState('');
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [newEmail, setNewEmail] = useState('');
  const [newTempPassword, setNewTempPassword] = useState('');
  const [deleteUserConfirm, setDeleteUserConfirm] = useState<string | null>(null);
  const [confirmUserTarget, setConfirmUserTarget] = useState<string | null>(null);
  const [resetPasswordTarget, setResetPasswordTarget] = useState<string | null>(null);
  const [selectedUser, setSelectedUser] = useState<CognitoUser | null>(null);

  // ---------- Data loading ----------

  const loadPools = useCallback(async () => {
    try {
      const res = await cognitoRequest('ListUserPools', { MaxResults: 60 });
      const list: UserPool[] = res.UserPools || [];
      setPools(list);

      const counts: Record<string, number> = {};
      for (const pool of list) {
        try {
          const usersRes = await cognitoRequest('ListUsers', { UserPoolId: pool.Id, Limit: 1 });
          counts[pool.Id] = usersRes.Users?.length ?? 0;
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

  async function createPool() {
    if (!newPoolName.trim()) return;
    try {
      await cognitoRequest('CreateUserPool', { PoolName: newPoolName.trim() });
      setShowCreatePool(false);
      setNewPoolName('');
      loadPools();
    } catch {
      // silently fail
    }
  }

  useEffect(() => { loadPools(); }, [loadPools]);

  const loadUsers = useCallback(async (poolId: string) => {
    setLoading(true);
    try {
      const res = await cognitoRequest('ListUsers', { UserPoolId: poolId, Limit: 60 });
      setUsers(res.Users || []);
    } catch {
      setUsers([]);
    }
    setLoading(false);
  }, []);

  // ---------- Actions ----------

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
        UserAttributes: [] as { Name: string; Value: string }[],
      };
      if (newEmail.trim()) {
        params.UserAttributes.push({ Name: 'email', Value: newEmail.trim() });
        params.UserAttributes.push({ Name: 'email_verified', Value: 'true' });
      }
      await cognitoRequest('AdminCreateUser', params);
      setShowCreateUser(false);
      setNewUsername('');
      setNewEmail('');
      setNewTempPassword('');
      loadUsers(selectedPool.Id);
      loadPools();
    } catch {
      // silently fail
    }
  }

  async function deleteUser(username: string) {
    if (!selectedPool) return;
    try {
      await cognitoRequest('AdminDeleteUser', {
        UserPoolId: selectedPool.Id,
        Username: username,
      });
      setDeleteUserConfirm(null);
      if (selectedUser?.Username === username) setSelectedUser(null);
      loadUsers(selectedPool.Id);
      loadPools();
    } catch {
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
      setConfirmUserTarget(null);
      loadUsers(selectedPool.Id);
    } catch {
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
      setResetPasswordTarget(null);
      loadUsers(selectedPool.Id);
    } catch {
      setResetPasswordTarget(null);
    }
  }

  // ---------- Derived state ----------

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

  // ---------- Render ----------

  return (
    <div class="cognito-layout">
      {/* Pool sidebar */}
      <div class="cognito-sidebar">
        <div class="cognito-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">User Pools</span>
            <div class="flex gap-2">
              <button class="btn btn-primary btn-sm" onClick={() => setShowCreatePool(true)}
                style="font-size:11px;padding:4px 10px">
                + New Pool
              </button>
              <button class="btn-icon" title="Refresh" onClick={loadPools}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <RefreshIcon />
              </button>
            </div>
          </div>
          <div class="search-wrap">
            <SearchIcon class="search-icon-pos" />
            <input class="input w-full search-input" placeholder="Filter pools..." value={poolSearch}
              onInput={(e) => setPoolSearch((e.target as HTMLInputElement).value)} />
          </div>
        </div>
        <div class="cognito-sidebar-list">
          {displayPools.map(pool => (
            <div key={pool.Id}
              class={`cognito-pool-item ${selectedPool?.Id === pool.Id ? 'active' : ''}`}
              onClick={() => selectPool(pool)}>
              <div class="pool-info">
                <div class="pool-name" title={pool.Name}>{pool.Name}</div>
                <div class="pool-id">{pool.Id}</div>
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
      <div class="cognito-main">
        {!selectedPool ? (
          <div class="cognito-empty-state">
            <div style="font-size:48px;opacity:0.3">Cognito</div>
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a user pool to manage users</div>
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">User pools are listed in the sidebar</div>
          </div>
        ) : selectedUser ? (
          <div>
            <div style="margin-bottom:16px">
              <button class="btn btn-ghost btn-sm" onClick={() => setSelectedUser(null)}>
                <ArrowLeftIcon /> Back to user list
              </button>
            </div>
            <div class="cognito-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedUser.Username}</h2>
                <div class="flex gap-2 items-center">
                  <span class={`cognito-status-badge ${statusBadgeClass(selectedUser.UserStatus)}`}>
                    {selectedUser.UserStatus}
                  </span>
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

            <div class="cognito-detail-card">
              <div class="cognito-detail-card-header">Attributes</div>
              <div class="cognito-detail-card-body">
                <table class="cognito-table">
                  <thead>
                    <tr><th>Name</th><th>Value</th></tr>
                  </thead>
                  <tbody>
                    {(selectedUser.Attributes || []).map(attr => (
                      <tr key={attr.Name}>
                        <td style="font-weight:600;font-size:13px">{attr.Name}</td>
                        <td style="font-family:var(--font-mono);font-size:12px">{attr.Value}</td>
                      </tr>
                    ))}
                    {(!selectedUser.Attributes || selectedUser.Attributes.length === 0) && (
                      <tr><td colSpan={2} style="text-align:center;color:var(--text-tertiary);padding:16px">No attributes</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            <div class="cognito-detail-card">
              <div class="cognito-detail-card-header">Details</div>
              <div class="cognito-detail-card-body">
                <div class="cognito-detail-grid">
                  <div><div class="label">Created</div><div>{formatDate(selectedUser.UserCreateDate)}</div></div>
                  <div><div class="label">Last Modified</div><div>{formatDate(selectedUser.UserLastModifiedDate)}</div></div>
                  <div><div class="label">Status</div><div>{selectedUser.UserStatus}</div></div>
                  <div><div class="label">Enabled</div><div>{selectedUser.Enabled ? 'Yes' : 'No'}</div></div>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div>
            <div class="cognito-header">
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

            <div class="search-wrap" style="margin-bottom:12px">
              <SearchIcon class="search-icon-pos" />
              <input class="input w-full search-input" placeholder="Filter users..." value={userSearch}
                onInput={(e) => setUserSearch((e.target as HTMLInputElement).value)} />
            </div>

            {loading ? (
              <div style="padding:32px;text-align:center;color:var(--text-tertiary)">Loading...</div>
            ) : (
              <table class="cognito-table">
                <thead>
                  <tr>
                    <th>Username</th><th>Email</th><th>Status</th><th>Created</th><th style="width:160px">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {displayUsers.map(user => (
                    <tr key={user.Username}>
                      <td>
                        <span style="cursor:pointer;color:var(--brand-blue);font-weight:600"
                          onClick={() => setSelectedUser(user)}>{user.Username}</span>
                      </td>
                      <td style="font-size:13px">{getUserAttr(user, 'email') || '--'}</td>
                      <td><span class={`cognito-status-badge ${statusBadgeClass(user.UserStatus)}`}>{user.UserStatus}</span></td>
                      <td style="font-size:12px;color:var(--text-secondary)">{formatDate(user.UserCreateDate)}</td>
                      <td>
                        <div class="flex gap-2">
                          {user.UserStatus !== 'CONFIRMED' && (
                            <button class="btn btn-ghost btn-sm" style="padding:2px 8px;font-size:11px"
                              onClick={() => setConfirmUserTarget(user.Username)}>Confirm</button>
                          )}
                          <button class="btn btn-ghost btn-sm" style="padding:2px 8px;font-size:11px"
                            onClick={() => setResetPasswordTarget(user.Username)}>Reset</button>
                          <button class="btn-icon" style="color:var(--error)"
                            onClick={() => setDeleteUserConfirm(user.Username)} title="Delete"><TrashIcon /></button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {displayUsers.length === 0 && (
                    <tr><td colSpan={5} style="text-align:center;padding:24px;color:var(--text-tertiary)">No users in this pool</td></tr>
                  )}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>

      {/* Create Pool Modal */}
      {showCreatePool && (
        <Modal title="Create User Pool" onClose={() => setShowCreatePool(false)}
          footer={<>
            <button class="btn btn-sm" onClick={() => setShowCreatePool(false)}>Cancel</button>
            <button class="btn btn-primary btn-sm" onClick={createPool} disabled={!newPoolName.trim()}>Create</button>
          </>}>
          <div class="field">
            <label class="label">Pool Name</label>
            <input class="input w-full" placeholder="e.g. my-app-users" value={newPoolName}
              onInput={(e) => setNewPoolName((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => { if (e.key === 'Enter') createPool(); }}
              autoFocus />
          </div>
        </Modal>
      )}

      {/* Create User Modal */}
      {showCreateUser && (
        <Modal title="Create User" onClose={() => setShowCreateUser(false)}
          footer={<>
            <button class="btn btn-ghost btn-sm" onClick={() => setShowCreateUser(false)}>Cancel</button>
            <button class="btn btn-primary btn-sm" onClick={createUser}>Create</button>
          </>}>
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
        <Modal title="Delete User" onClose={() => setDeleteUserConfirm(null)}
          footer={<>
            <button class="btn btn-ghost btn-sm" onClick={() => setDeleteUserConfirm(null)}>Cancel</button>
            <button class="btn btn-danger btn-sm" onClick={() => deleteUser(deleteUserConfirm)}>Delete</button>
          </>}>
          <p>Are you sure you want to delete user <strong>{deleteUserConfirm}</strong>? This cannot be undone.</p>
        </Modal>
      )}

      {/* Confirm User */}
      {confirmUserTarget && (
        <Modal title="Confirm User" onClose={() => setConfirmUserTarget(null)}
          footer={<>
            <button class="btn btn-ghost btn-sm" onClick={() => setConfirmUserTarget(null)}>Cancel</button>
            <button class="btn btn-primary btn-sm" onClick={() => confirmUser(confirmUserTarget)}>Confirm</button>
          </>}>
          <p>Confirm user <strong>{confirmUserTarget}</strong>? This will mark them as verified.</p>
        </Modal>
      )}

      {/* Reset Password */}
      {resetPasswordTarget && (
        <Modal title="Reset Password" onClose={() => setResetPasswordTarget(null)}
          footer={<>
            <button class="btn btn-ghost btn-sm" onClick={() => setResetPasswordTarget(null)}>Cancel</button>
            <button class="btn btn-danger btn-sm" onClick={() => resetPassword(resetPasswordTarget)}>Reset</button>
          </>}>
          <p>Reset password for <strong>{resetPasswordTarget}</strong>? They will need to set a new password.</p>
        </Modal>
      )}
    </div>
  );
}
