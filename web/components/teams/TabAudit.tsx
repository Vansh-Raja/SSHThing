"use client";

import { formatTime } from "./utils";
import type { TeamAuditEvent } from "./types";

type TabAuditProps = {
  events: TeamAuditEvent[];
  canView: boolean;
};

export default function TabAudit({ events, canView }: TabAuditProps) {
  return (
    <>
      <div className="tab-toolbar">
        <h2 className="tab-toolbar__title">Audit</h2>
      </div>

      <div className="tab-content">
        {!canView ? (
          <div className="empty-state">
            <div className="empty-state__title">Admin only</div>
            <p className="empty-state__body">
              Owners and admins can see credential reveal and delete history.
            </p>
          </div>
        ) : events.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No audited events yet</div>
            <p className="empty-state__body">
              Credential reveals and deletes will show here with actor, target,
              and timestamp.
            </p>
          </div>
        ) : (
          <div>
            {events.map((event) => (
              <div key={event.id} className="data-row">
                <div className="data-row__primary">
                  <span className="data-row__title">{event.summary}</span>
                  <span className="data-row__meta">
                    {event.actorDisplayName}
                    {event.targetDisplayName ? ` → ${event.targetDisplayName}` : ""}
                    {event.metadata?.hostLabel ? ` · ${event.metadata.hostLabel}` : ""}
                    {event.metadata?.credentialType
                      ? ` · ${event.metadata.credentialType}`
                      : ""}
                  </span>
                </div>
                <div className="data-row__trail">
                  <span className="chip chip--muted">
                    {formatTime(event.createdAt)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}
