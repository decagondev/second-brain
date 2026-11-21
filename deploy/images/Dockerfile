# One image per cell service (sb-selfhost(7); SB-1601): a static Go binary built by
# `make images`, on the distroless static base pinned by digest, as the non-root user,
# with no shell. The same image runs on Cloud Run and on a self-hosted machine.
FROM gcr.io/distroless/static-debian12@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
ARG SVC
COPY bin/${SVC} /sb/service
COPY web /sb/web
USER 65532:65532
ENTRYPOINT ["/sb/service"]
