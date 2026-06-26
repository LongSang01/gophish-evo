var TEMPLATE_TAGS = [{
        id: 1,
        name: 'RId',
        description: '收件人的唯一标识。'
    },
    {
        id: 2,
        name: 'FullName',
        description: '收件人的姓名。'
    },
    {
        id: 3,
        name: 'Position',
        description: '收件人的职位。'
    },
    {
        id: 4,
        name: 'From',
        description: '邮件的发件地址。'
    },
    {
        id: 5,
        name: 'TrackingURL',
        description: '用于追踪邮件是否被打开的URL。'
    },
    {
        id: 6,
        name: 'Tracker',
        description: '添加隐藏追踪图片的HTML标签（推荐替代TrackingURL）。'
    },
    {
        id: 7,
        name: 'URL',
        description: '您的Gophish监听器URL。'
    },
    {
        id: 8,
        name: 'BaseURL',
        description: '去除路径和rid参数的基础URL，适用于静态文件链接。'
    },
    {
        id: 9,
        name: 'QRCode',
        description: '链接到钓鱼URL的二维码图片。'
    }
];

var textTestCallback = function (range) {
    if (!range.collapsed) {
        return null;
    }

    return CKEDITOR.plugins.textMatch.match(range, matchCallback);
}

var matchCallback = function (text, offset) {
    var pattern = /\{{2}\.?([A-z]|\})*$/,
        match = text.slice(0, offset)
        .match(pattern);

    if (!match) {
        return null;
    }

    return {
        start: match.index,
        end: offset
    };
}

/**
 * 
 * @param {regex} matchInfo - The matched text object
 * @param {function} callback - The callback to execute with the matched data
 */
var dataCallback = function (matchInfo, callback) {
    var data = TEMPLATE_TAGS.filter(function (item) {
        var itemName = '{{.' + item.name.toLowerCase() + '}}';
        return itemName.indexOf(matchInfo.query.toLowerCase()) == 0;
    });

    callback(data);
}

/**
 * 
 * @param {CKEditor} editor - The CKEditor instance.
 * 
 * Installs the autocomplete plugin to the CKEditor.
 */
var setupAutocomplete = function (editor) {
    editor.on('instanceReady', function (evt) {
        var itemTemplate = '<li data-id="{id}">' +
            '<div><strong class="item-title">{name}</strong></div>' +
            '<div><i>{description}</i></div>' +
            '</li>',
            outputTemplate = '[[.{name}]]';

        var autocomplete = new CKEDITOR.plugins.autocomplete(evt.editor, {
            textTestCallback: textTestCallback,
            dataCallback: dataCallback,
            itemTemplate: itemTemplate,
            outputTemplate: outputTemplate
        });

        // We have to use brackets for the output template tag and 
        // then manually replace them due to the way CKEditor's 
        // templating works.
        autocomplete.getHtmlToInsert = function (item) {
            var parsedTemplate = this.outputTemplate.output(item);
            parsedTemplate = parsedTemplate.replace("[[", "{{").replace("]]", "}}")
            return parsedTemplate
        }
    });
}