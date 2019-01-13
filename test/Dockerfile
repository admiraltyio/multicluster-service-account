FROM golang:1.10.3

COPY test/install_kubebuilder.sh .
RUN	[ "sh", "install_kubebuilder.sh" ]
ENV KUBEBUILDER_ASSETS=/kubebuilder/bin

WORKDIR /go/src/admiralty.io/multicluster-service-account

COPY vendor vendor
COPY pkg pkg
COPY cmd cmd
COPY examples examples

COPY test/test.sh .
ENTRYPOINT [ "sh", "test.sh" ]
