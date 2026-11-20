// Department spaces (SB-1004; sb-permissions(7), Spaces, labels and barriers, exactly).
// Space rules only forbid: the base read permit stays the only permit on read, and the
// source's live ACL is still checked after (LAW-19). Each rule applies only in a cell with
// spaces, where the kernel sets context.space; any missing fact there refuses (LAW-03).

// 0. A cell with spaces gives every fact, or the read is refused.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space &&
  !(context has member_spaces && context has barred_spaces && context has barred_groups &&
    resource has label && resource has spaces && resource has owners)
};

// 1. Membership: an ask in a space reads only what that space holds, and only for its members.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space && context has member_spaces && resource has spaces &&
  context.space != "company" &&
  !(resource.spaces.contains(context.space) && context.member_spaces.contains(context.space))
};

// 2. Label: only public and internal reach people outside the spaces that hold a chunk.
// Any other label, known or not, is treated as restricted (LAW-20).
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space && context has member_spaces && resource has label && resource has spaces &&
  !(resource.label == "public" || resource.label == "internal") &&
  !resource.spaces.containsAny(context.member_spaces)
};

// 3. Barrier: nothing held by a space across a barrier from the asker (LAW-22).
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has barred_spaces && resource has spaces &&
  resource.spaces.containsAny(context.barred_spaces)
};

// 3b. Ownership: nothing owned at the source by someone in a group across a barrier from
// the asker, even company-wide and even if shared in the source (SB-1203).
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has barred_groups && resource has owners &&
  resource.owners.containsAny(context.barred_groups)
};

// 4. Actions: no tool call or proposal on an item held or owned across a barrier from the
// requester; in a cell with barriers, a target the cell can't place refuses (SB-1203).
forbid (principal, action in [Action::"call_tool", Action::"propose"], resource)
when {
  context has barred_groups &&
  !(context has barred_spaces && context has target_spaces && context has target_owners)
};

forbid (principal, action in [Action::"call_tool", Action::"propose"], resource)
when {
  context has barred_spaces && context has target_spaces &&
  context.target_spaces.containsAny(context.barred_spaces)
};

forbid (principal, action in [Action::"call_tool", Action::"propose"], resource)
when {
  context has barred_groups && context has target_owners &&
  context.target_owners.containsAny(context.barred_groups)
};

// 5. Approvals: nobody approves an action whose requester is across a barrier from them.
forbid (principal, action == Action::"approve", resource is ProposedAction)
when { context has barred_groups && !(context has requester_groups) };

forbid (principal, action == Action::"approve", resource is ProposedAction)
when {
  context has barred_groups && context has requester_groups &&
  context.requester_groups.containsAny(context.barred_groups)
};
