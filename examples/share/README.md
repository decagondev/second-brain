# The example share

A small file share for the self-hosted demo (`docs: folder`, sb-connector(3); SB-1606). Every
document and name in it is made up. `make selfhost` mounts it read-only into the cell.

Who may read each file is in the ACL file beside it (`FILE.acl`) or in `.acl` for its folder:

- `operations/` is readable by everyone at acme.example.
- `finance/q3-forecast.md` only by ann and the finance lead, and it is labelled confidential.
- `people/` is readable only by the people team, so ann's questions never see it.
- `README.md` itself has no ACL file, so nobody is allowed it: a file with no ACL is
  Unknown, never allowed.
