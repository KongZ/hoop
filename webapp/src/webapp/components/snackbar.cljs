(ns webapp.components.snackbar
  (:require
   [re-frame.core :as rf]))

(defmulti level-icon identity)
(defmethod level-icon :error [_] "/icons/icon-important-red.svg")
(defmethod level-icon :success [_] "/icons/icon-check-green.svg")
(defmethod level-icon :info [_] "/icons/icon-information-white.svg")
(defmethod level-icon :default [_] "/icons/icon-information-white.svg")

(defmulti markup identity)
(defmethod markup :shown [_ state]
  (js/setTimeout #(rf/dispatch [:hide-snackbar]) 10000)
  [:div {:class (str "flex align-center z-50 fixed max-w-xs top-8 right-8 p-regular bg-gray-800 "
                     "font-light text-gray-100 leading-5 rounded-lg shadow-lg animate-appear-right whitespace-normal")}
   [:figure {:class "flex-shrink-0 w-6 mr-regular"}
    [:img {:src (level-icon (:level state))}]]
   [:div.flex-shrink
    [:small {:class "whitespace-normal"}
     (:text state)]]
   [:figure.flex-shrink-0.w-6.ml-regular.cursor-pointer
    {:on-click #(rf/dispatch [:hide-snackbar])}
    [:img {:src "/icons/icon-close-white.svg"}]]])

(defmethod markup :default [_] nil)

(defn snackbar []
  (let [state @(rf/subscribe [:snackbar])]
    (markup (:status state) state)))
