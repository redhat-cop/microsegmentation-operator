FROM registry.access.redhat.com/ubi7-dev-preview/ubi-minimal:7.6

ENV OPERATOR=/usr/local/bin/microsegmentation-operator \
    USER_UID=1001 \
    USER_NAME=microsegmentation-operator

# install operator binary
COPY build/_output/bin/microsegmentation-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
