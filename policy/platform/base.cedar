// Platform base policies (sb-policy(5)). Default deny; forbid beats permit.

// Reads are scoped to the principal's tenant. The source's live ACL must also allow (ADR-0002).
permit (principal, action == Action::"read", resource is Chunk)
when { principal.tenant == resource.tenant };

permit (principal, action == Action::"read", resource is Document)
when { principal.tenant == resource.tenant };

// Everyone in a tenant may use their own cell's ask agent.
permit (principal, action == Action::"use_agent", resource == Agent::"ask")
when { principal.tenant == resource.tenant };

// draft-reply, where the tenant's manifest enables it (SB-409); tenant policies may narrow
// it further (by group or by ticket project). It drafts only: a person always sends.
permit (principal, action == Action::"use_agent", resource == Agent::"draft-reply")
when { principal.tenant == resource.tenant && resource has enabled && resource.enabled };

// Nobody approves their own action (LAW-15).
forbid (principal, action == Action::"approve", resource is ProposedAction)
when { resource.requester == principal };

// Never approve across tenants (LAW-02).
forbid (principal, action == Action::"approve", resource is ProposedAction)
unless { principal.tenant == resource.tenant };

// Admin only on your own cell.
forbid (principal, action == Action::"admin", resource is Cell)
unless { principal.tenant == resource.tenant };

// Tools from the release's registry (SB-405): any user of the tenant may call a tool that
// isn't consequential; the tool bus still checks kill switches and audits first.
permit (principal, action == Action::"call_tool", resource is Tool)
when { principal.tenant == resource.tenant && !resource.consequential };

// Proposing an action for the tenant's enabled workflow (SB-406). It only drafts: running
// it needs an approval by someone else in the workflow's approver group.
permit (principal, action == Action::"propose", resource is Workflow)
when { principal.tenant == resource.tenant };

// Approvers are the workflow's approver group (sb-approvals(7)); the forbids above still
// stop self-approval and approval across tenants.
permit (principal, action == Action::"approve", resource is ProposedAction)
when { principal.tenant == resource.tenant && principal in resource.approvers };

