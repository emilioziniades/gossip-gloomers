(defn msgs-success [msgs] (< msgs 20))
(defn mid-latency-success [lat] (< lat 1000))
(defn max-latency-success [lat] (< lat 2000))
(fn
  [data]
  (let [msgs-per-op (-> data :net :servers :msgs-per-op)
        median-latency (-> data :workload :stable-latencies (get 0.5))
        max-latency (-> data :workload :stable-latencies (get 1))]
    {:metrics {:messages-per-operation
               {:value msgs-per-op
                :less-than-20 (msgs-success msgs-per-op)}
               :median-latency
               {:value median-latency
                :less-than-1000ms (mid-latency-success median-latency)}
               :maximum-latency
               {:value max-latency
                :less-than-2000ms (max-latency-success max-latency)}}
     :success (every?
               true?
               [(msgs-success msgs-per-op)
                (mid-latency-success median-latency)
                (max-latency-success max-latency)])}))
