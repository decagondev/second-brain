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

// Department agents (SB-1102), where the manifest enables them: read-only, and the cell
// runs each only in a space of its own template (sb-agents(7), Department agents).
permit (principal, action == Action::"use_agent", resource is Agent)
when {
  principal.tenant == resource.tenant && resource has enabled && resource.enabled && !resource.consequential &&
  (resource.name == "incident-summary" || resource.name == "spec-digest" || resource.name == "close-checklist" || resource.name == "runbook-answer")
};

// Stewards (SB-1201): a member of one of a space's stewards groups may propose a change to
// that space; the kernel puts the asker's groups, as group:NAME, in context.groups.
permit (principal, action == Action::"propose_space", resource is SpaceConfig)
when { principal.tenant == resource.tenant && context has groups && resource.stewards.containsAny(context.groups) };

// Stewards run their space's quarterly access review (SB-1202).
permit (principal, action == Action::"review_space", resource is SpaceConfig)
when { principal.tenant == resource.tenant && context has groups && resource.stewards.containsAny(context.groups) };

// Operators approve stewards' proposals, never their own (LAW-15).
permit (principal in Group::"sb-operators", action == Action::"approve_space", resource is SpaceProposal)
when { principal.tenant == resource.tenant };

forbid (principal, action == Action::"approve_space", resource is SpaceProposal)
when { resource.requester == principal };

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

// The operator role (SB-603): members of the tenant's own sb-operators group administer
// their own cell. Groups come from the cell's directory, so this never reaches another
// tenant; the forbid above is the second guard.
permit (principal in Group::"sb-operators", action == Action::"admin", resource is Cell)
when { principal.tenant == resource.tenant };


// Factory operators (SB-702): the sb-factory-operators group of the factory's own tenant,
// on custody records the factory holds. Custody records always carry the factory's tenant;
// the kernel never fills it in for them.
permit (
  principal in Group::"sb-factory-operators",
  action in [Action::"view_custody", Action::"plan_handoff", Action::"approve_handoff", Action::"execute_handoff", Action::"abort_handoff"],
  resource is CellCustody
)
when { principal.tenant == resource.tenant };

// Tenant owners (SB-702): the sb-owners group of the tenant's own IdP, on their own cell.
permit (
  principal in Group::"sb-owners",
  action in [Action::"view_custody", Action::"approve_handoff", Action::"detach"],
  resource is Cell
)
when { principal.tenant == resource.tenant };

// Nobody approves their own hand-off plan: the factory side needs a second operator.
forbid (principal, action == Action::"approve_handoff", resource)
when { context has planner && context.planner == principal };

// Neither role reads content outside its own tenant (LAW-02); they add no read permit.
forbid (principal in Group::"sb-factory-operators", action == Action::"read", resource)
unless { principal.tenant == resource.tenant };

forbid (principal in Group::"sb-owners", action == Action::"read", resource)
unless { principal.tenant == resource.tenant };
