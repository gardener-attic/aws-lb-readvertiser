# SPDX-FileCopyrightText: 2017 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM gcr.io/distroless/static-debian11:nonroot

ADD ./bin/rel/aws-lb-readvertiser /aws-lb-readvertiser

WORKDIR /

ENTRYPOINT ["/aws-lb-readvertiser"]
