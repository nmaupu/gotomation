FROM gcr.io/distroless/base

ARG GOTOMATION_VERSION
ARG GOTOMATION_BIN_DIR
COPY ${GOTOMATION_BIN_DIR}/gotomation-linux_amd64-${GOTOMATION_VERSION} /bin/gotomation

USER 999

CMD ["/bin/gotomation"]
