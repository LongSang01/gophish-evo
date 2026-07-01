function t(r){if(!r||r==="0001-01-01T00:00:00Z"||r.startsWith("0001-01-01"))return"-";try{return new Date(r).toLocaleString("zh-CN")}catch{return r}}export{t as f};
