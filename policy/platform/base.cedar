// Platform base policies (sb-policy(5)). Default deny; forbid beats permit.

// Reads are scoped to the principal's tenant. The source's live ACL must also allow (ADR-0002).
permit (principal, action == Action::"read", resource is Chunk)
when { principal.tenant == resource.tenant };

permit (principal, action == Action::"read", resource is Document)
when { principal.tenant == resource.tenant };

// Everyone in a tenant may use their own cell's ask agent.
permit (principal, action == Action::"use_agent", resource == Agent::"ask")
when { principal.tenant == resource.tenant };

// Nobody approves their own action (LAW-15).
forbid (principal, action == Action::"approve", resource is ProposedAction)
when { resource.requester == principal };

// Never approve across tenants (LAW-02).
forbid (principal, action == Action::"approve", resource is ProposedAction)
unless { principal.tenant == resource.tenant };

// Admin only on your own cell.
forbid (principal, action == Action::"admin", resource is Cell)
unless { principal.tenant == resource.tenant };
