// Department spaces (SB-1004; sb-permissions(7), Spaces, labels and barriers, exactly).
// Space rules only forbid: the base read permit stays the only permit on read, and the
// source's live ACL is still checked after (LAW-19). Each rule applies only in a cell with
// spaces, where the kernel sets context.space; any missing fact there refuses (LAW-03).

// 0. A cell with spaces gives every fact, or the read is refused.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space &&
  !(context has member_spaces && context has barred_spaces && resource has label && resource has spaces)
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
