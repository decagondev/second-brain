// Tenant policy for acme-logistics. Tenant policies can only narrow platform policy.

// Only IT leads approve ticket updates.
permit (principal in Group::"grp_it_leads", action == Action::"approve", resource is ProposedAction)
when { resource.workflow == "ticket-update" && principal.tenant == resource.tenant };

// Contractors can ask, but not draft.
forbid (principal in Group::"grp_contractors", action == Action::"use_agent", resource == Agent::"draft-reply");

// Finance tickets are drafted only by finance staff.
forbid (principal, action == Action::"use_agent", resource == Agent::"draft-reply")
when { context has target_project && context.target_project == "FIN" && !(principal in Group::"grp_finance") };
