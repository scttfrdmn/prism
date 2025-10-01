"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const base_input_js_1 = require("../input/base-input.js");
const styles_selectors_js_1 = require("../../../slider/styles.selectors.js");
class SliderWrapper extends base_input_js_1.default {
    findNativeInput() {
        return this.findByClassName(styles_selectors_js_1.default.thumb);
    }
}
exports.default = SliderWrapper;
SliderWrapper.rootSelector = styles_selectors_js_1.default.root;
//# sourceMappingURL=index.js.map