/**
 * Shared campaign type utilities used by both the campaign list and detail views.
 */

/** Display text for each campaign source type. */
export const SOURCE_TYPE_TEXT: Record<string, string> = {
  email: "邮件钓鱼",
  client: "钓鱼客户端",
  page: "固定钓鱼页面",
};

/** Tag color for each campaign source type. */
export const SOURCE_TYPE_COLOR: Record<string, string> = {
  email: "blue",
  client: "purple",
  page: "orange",
};

/** Human-readable label for a source type, falling back to the raw value. */
export function sourceTypeText(type: string): string {
  return SOURCE_TYPE_TEXT[type] || type;
}

/** Ant-Design tag color for a source type. */
export function sourceTypeColor(type: string): string {
  return SOURCE_TYPE_COLOR[type] || "default";
}

/** Default report config for client/page type campaigns. */
export const DEFAULT_REPORT_CONFIG = {
  dedup_key: "mac",
  fields: [
    {
      key: "ip",
      label: "IP地址",
      type: "ip",
      required: false,
      placeholder: "",
    },
    {
      key: "mac",
      label: "MAC地址",
      type: "mac",
      required: false,
      placeholder: "",
    },
    {
      key: "username",
      label: "用户名",
      type: "username",
      required: false,
      placeholder: "",
    },
    {
      key: "hostname",
      label: "主机名",
      type: "hostname",
      required: false,
      placeholder: "",
    },
  ],
};

/** Default form data for creating or duplicating a campaign. */
export function defaultCampaignForm() {
  return {
    source_type: "email",
    name: "",
    template_id: null as number | null,
    group_ids: [] as number[],
    smtp_ids: [] as number[],
    page_id: null as number | null,
    url: "",
    page_path: "",
    launch_date: null as any,
    send_by_date: null as any,
    report_config: JSON.parse(JSON.stringify(DEFAULT_REPORT_CONFIG)),
  };
}

/**
 * Populate a form object from an existing campaign (for duplicate / copy).
 * Pass `dayjs` as the second argument to convert date strings for date-picker
 * components; omit it to keep raw ISO strings.
 */
export function fillFormFromCampaign(
  form: ReturnType<typeof defaultCampaignForm>,
  campaign: any,
  dayjsFn?: (v: string) => any,
) {
  form.source_type = campaign.source_type || "email";
  form.name = `${campaign.name} - 副本`;
  form.template_id = campaign.template?.id || null;
  form.group_ids = campaign.groups?.map((g: any) => g.id) || [];
  form.smtp_ids = campaign.smtps?.map((s: any) => s.id) || [];
  form.page_id = campaign.page?.id || null;
  form.page_path = "";
  form.url = campaign.url || "";
  // For page campaigns, split stored full URL back into base URL + path.
  if (campaign.source_type === "page" && campaign.url) {
    try {
      const u = new URL(campaign.url);
      form.url = u.origin;
      form.page_path = u.pathname;
    } catch {
      form.url = campaign.url;
    }
  }
  form.launch_date =
    campaign.launch_date && dayjsFn
      ? dayjsFn(campaign.launch_date)
      : campaign.launch_date || null;
  form.send_by_date =
    campaign.send_by_date && dayjsFn
      ? dayjsFn(campaign.send_by_date)
      : campaign.send_by_date || null;
  form.report_config = {
    dedup_key: campaign.report_config?.dedup_key || "mac",
    fields: (campaign.report_config?.fields || []).map((f: any) => ({ ...f })),
  };
}
