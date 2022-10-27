FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

COPY bin/probe /stackrox/

EXPOSE 7070

ENTRYPOINT ["/stackrox/probe"]

CMD ["start"]

LABEL name="probe" \
	vendor="Red Hat" \
	version="0.0.1" \
	summary="Blackbox monitoring probe for ACS Fleet Manager" \
	description="Blackbox monitoring probe for the Red Hat Advanced Cluster Security Fleet Manager"
