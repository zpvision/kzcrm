export type User={id:string;email:string;name:string;role:'admin'|'partner';csrfToken:string};
export type Client={id:string;companyName:string;taxId?:string;email:string;contactName?:string;phone?:string;comment?:string;count?:number;finKoperExpires?:string;whatsAppEnabled:boolean;whatsAppEnabledAt?:string;whatsAppExpires?:string;finKoperStatus:Status;whatsAppStatus:Status;version:number;archivedAt?:string};
export type Status='unpaid'|'pending'|'completed';
export type CountRow={clientId:string;companyName:string;email:string;taxId?:string;previous?:number;count?:number;version?:number};
